package utils

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// PartSuffixLen caps a split archive at 26^N parts (3 → 17576).
const PartSuffixLen = 3

// ParseSize parses sizes like "100MB", "2 GiB", "512K", "1024". Both KB and KiB
// resolve to 1024 for consistency with FormatBytes output.
func ParseSize(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty size")
	}

	end := 0
	for end < len(s) {
		c := s[end]
		if (c >= '0' && c <= '9') || c == '.' {
			end++
			continue
		}
		break
	}
	if end == 0 {
		return 0, fmt.Errorf("invalid size %q: no number", s)
	}

	numPart := s[:end]
	unitPart := strings.TrimSpace(strings.ToUpper(s[end:]))

	num, err := strconv.ParseFloat(numPart, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size %q: %w", s, err)
	}
	if num < 0 {
		return 0, fmt.Errorf("invalid size %q: negative", s)
	}

	var mult float64
	switch unitPart {
	case "", "B":
		mult = 1
	case "K", "KB", "KIB":
		mult = 1024
	case "M", "MB", "MIB":
		mult = 1024 * 1024
	case "G", "GB", "GIB":
		mult = 1024 * 1024 * 1024
	case "T", "TB", "TIB":
		mult = 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("invalid size %q: unknown unit %q", s, unitPart)
	}

	bytes := int64(num * mult)
	if bytes <= 0 {
		return 0, fmt.Errorf("invalid size %q: must be > 0", s)
	}
	return bytes, nil
}

// PartSuffix maps a zero-based index to a base-26 suffix (0→"aaa", 26→"aba").
func PartSuffix(index int) (string, error) {
	if index < 0 {
		return "", fmt.Errorf("part index %d is negative", index)
	}
	max := 1
	for i := 0; i < PartSuffixLen; i++ {
		max *= 26
	}
	if index >= max {
		return "", fmt.Errorf("part index %d exceeds maximum %d (chunk size too small for archive)", index, max-1)
	}

	out := make([]byte, PartSuffixLen)
	for i := PartSuffixLen - 1; i >= 0; i-- {
		out[i] = 'a' + byte(index%26)
		index /= 26
	}
	return string(out), nil
}

// SplitFile cuts srcPath into "<srcPath>.part-aaa" parts and removes the source
// on success. Always produces at least one part, even if srcPath fits in one chunk.
func SplitFile(srcPath string, chunkSize int64) ([]string, error) {
	if chunkSize <= 0 {
		return nil, fmt.Errorf("chunk size must be > 0, got %d", chunkSize)
	}

	src, err := os.Open(srcPath) //nolint:gosec // srcPath comes from snapr's own pipeline
	if err != nil {
		return nil, fmt.Errorf("open source: %w", err)
	}
	defer func() { _ = src.Close() }()

	var parts []string
	buf := make([]byte, 32*1024)
	idx := 0

	for {
		suffix, err := PartSuffix(idx)
		if err != nil {
			cleanupParts(parts)
			return nil, err
		}
		partPath := srcPath + ".part-" + suffix

		written, copyErr := writePart(src, partPath, chunkSize, buf)
		if copyErr != nil {
			cleanupParts(parts)
			cleanupParts([]string{partPath})
			return nil, copyErr
		}
		if written == 0 {
			_ = os.Remove(partPath)
			break
		}
		parts = append(parts, partPath)
		if written < chunkSize {
			break
		}
		idx++
	}

	if len(parts) == 0 {
		return nil, fmt.Errorf("source file %s is empty", srcPath)
	}

	if err := src.Close(); err != nil {
		cleanupParts(parts)
		return nil, fmt.Errorf("close source: %w", err)
	}
	if err := os.Remove(srcPath); err != nil {
		cleanupParts(parts)
		return nil, fmt.Errorf("remove source: %w", err)
	}
	return parts, nil
}

func writePart(src io.Reader, partPath string, chunkSize int64, buf []byte) (int64, error) {
	out, err := os.Create(partPath) //nolint:gosec // partPath built from snapr's own archive path
	if err != nil {
		return 0, fmt.Errorf("create part %s: %w", partPath, err)
	}

	var written int64
	closed := false
	defer func() {
		if !closed {
			_ = out.Close()
		}
	}()

	for written < chunkSize {
		toRead := int64(len(buf))
		if remaining := chunkSize - written; remaining < toRead {
			toRead = remaining
		}
		n, readErr := src.Read(buf[:toRead])
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				return written, fmt.Errorf("write part %s: %w", partPath, werr)
			}
			written += int64(n)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return written, fmt.Errorf("read source: %w", readErr)
		}
	}

	if err := out.Close(); err != nil {
		return written, fmt.Errorf("close part %s: %w", partPath, err)
	}
	closed = true
	return written, nil
}

func cleanupParts(parts []string) {
	for _, p := range parts {
		_ = os.Remove(p)
	}
}

// IsPartName returns the base name and true if name ends with ".part-XXX".
func IsPartName(name string) (string, bool) {
	const marker = ".part-"
	idx := strings.LastIndex(name, marker)
	if idx == -1 {
		return "", false
	}
	suffix := name[idx+len(marker):]
	if len(suffix) != PartSuffixLen {
		return "", false
	}
	for i := 0; i < len(suffix); i++ {
		c := suffix[i]
		if c < 'a' || c > 'z' {
			return "", false
		}
	}
	return name[:idx], true
}
