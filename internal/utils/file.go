package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func IsLocalUpToDate(localPath string, remoteSize int64, remoteMTime time.Time) bool {
	stat, err := os.Stat(localPath)
	if err != nil {
		return false
	}
	if stat.Size() != remoteSize {
		return false
	}
	if !remoteMTime.IsZero() && stat.ModTime().Before(remoteMTime) {
		return false
	}
	return true
}

func SafeJoin(base, rel string) (string, error) {
	cleanBase, err := filepath.Abs(filepath.Clean(base))
	if err != nil {
		return "", fmt.Errorf("invalid base path: %w", err)
	}

	joined := filepath.Join(cleanBase, rel)
	cleanJoined, err := filepath.Abs(joined)
	if err != nil {
		return "", fmt.Errorf("invalid joined path: %w", err)
	}

	if cleanJoined != cleanBase && !strings.HasPrefix(cleanJoined, cleanBase+string(os.PathSeparator)) {
		return "", fmt.Errorf("path traversal detected: %q escapes %q", rel, base)
	}

	return joined, nil
}

func IsExcluded(path string, excludes []string) bool {
	if len(excludes) == 0 {
		return false
	}

	normalizedPath := filepath.ToSlash(path)
	filename := filepath.Base(normalizedPath)

	for _, pattern := range excludes {
		if pattern == "" {
			continue
		}

		normalizedPattern := filepath.ToSlash(strings.TrimSpace(pattern))

		if strings.HasSuffix(normalizedPattern, "/") {
			dirPattern := strings.TrimSuffix(normalizedPattern, "/")

			if matched, _ := filepath.Match(dirPattern, normalizedPath); matched {
				return true
			}

			if strings.HasPrefix(normalizedPath+"/", dirPattern+"/") {
				return true
			}

			if strings.Contains(dirPattern, "**/") {
				if matchesDeepPattern(normalizedPath, dirPattern, true) {
					return true
				}
			}
			continue
		}

		if strings.Contains(normalizedPattern, "**/") {
			if matchesDeepPattern(normalizedPath, normalizedPattern, false) {
				return true
			}
			if matchesDeepPattern(filename, normalizedPattern, false) {
				return true
			}
			continue
		}

		if matched, _ := filepath.Match(normalizedPattern, normalizedPath); matched {
			return true
		}

		if matched, _ := filepath.Match(normalizedPattern, filename); matched {
			return true
		}

		if !strings.ContainsAny(normalizedPattern, "*?[") {
			pathParts := strings.Split(normalizedPath, "/")
			for _, part := range pathParts {
				if part == normalizedPattern {
					return true
				}
			}
		}
	}

	return false
}

func matchesDeepPattern(path, pattern string, isDirPattern bool) bool {
	if strings.HasPrefix(pattern, "**/") {
		suffix := strings.TrimPrefix(pattern, "**/")

		if strings.HasSuffix(suffix, "/**") {
			dirName := strings.TrimSuffix(suffix, "/**")
			pathParts := strings.Split(path, "/")

			for i, part := range pathParts {
				if matched, _ := filepath.Match(dirName, part); matched {
					if isDirPattern {
						return true
					}
					if i < len(pathParts)-1 {
						return true
					}
				}
			}
		} else {
			pathParts := strings.Split(path, "/")
			for _, part := range pathParts {
				if matched, _ := filepath.Match(suffix, part); matched {
					return true
				}
			}
		}
	}

	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		if strings.HasPrefix(path, prefix+"/") || path == prefix {
			return true
		}
	}

	return false
}
