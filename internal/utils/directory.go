package utils

import (
	"io/fs"
	"os"
	"path/filepath"
	"syscall"

	"github.com/rs/zerolog/log"
)

func RemoveEmptyDirs(root string) {
	_ = filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil || info == nil || !info.IsDir() || path == root {
			return nil //nolint:nilerr
		}
		entries, readErr := os.ReadDir(path)
		if readErr != nil {
			return nil //nolint:nilerr
		}
		if len(entries) == 0 {
			_ = os.Remove(path) //nolint:gosec // cleanup of own walked tree
		}
		return nil
	})
}

func CalculateDirectorySize(dirPath string) int64 {
	var totalSize int64
	visited := make(map[uint64]bool)

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Warn().Err(err).Str("path", path).Msg("Error accessing file during size calculation")
			return nil
		}

		info, err := d.Info()
		if err != nil {
			log.Warn().Err(err).Str("path", path).Msg("Error getting file info")
			return nil
		}

		if info.Mode()&os.ModeSymlink != 0 {
			targetPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				log.Warn().Err(err).Str("path", path).Msg("Error evaluating symlink")
				return nil
			}

			targetInfo, err := os.Stat(targetPath)
			if err != nil {
				log.Warn().Err(err).Str("path", path).Str("target", targetPath).Msg("Error accessing symlink target")
				return nil
			}

			if targetInfo.IsDir() {
				symlinkSize := CalculateDirectorySize(targetPath)
				if symlinkSize > 0 {
					totalSize += symlinkSize
				}
				return filepath.SkipDir
			}

			if targetInfo.Mode().IsRegular() {
				if stat, ok := targetInfo.Sys().(*syscall.Stat_t); ok {
					if !visited[stat.Ino] {
						visited[stat.Ino] = true
						totalSize += targetInfo.Size()
					}
				} else {
					totalSize += targetInfo.Size()
				}
			}

			return nil
		}

		if d.IsDir() {
			return nil
		}

		if info.Mode().IsRegular() {
			if stat, ok := info.Sys().(*syscall.Stat_t); ok {
				if visited[stat.Ino] {
					return nil
				}
				visited[stat.Ino] = true
			}
			totalSize += info.Size()
		}

		return nil
	})

	if err != nil {
		return -1
	}

	return totalSize
}
