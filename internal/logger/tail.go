package logger

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"time"
)

func TailFile(path string, n int) ([][]byte, error) {
	if n <= 0 {
		return nil, nil
	}
	f, err := os.Open(path) //nolint:gosec // path comes from server config, callers validate it
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	stat, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", path, err)
	}
	size := stat.Size()
	if size == 0 {
		return nil, nil
	}

	const chunk = 64 * 1024
	var (
		buf    []byte
		lines  [][]byte
		offset = size
	)

	for offset > 0 {
		readSize := int64(chunk)
		if offset < readSize {
			readSize = offset
		}
		offset -= readSize

		tmp := make([]byte, readSize)
		if _, err := f.ReadAt(tmp, offset); err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		buf = append(tmp, buf...) //nolint:gocritic // need prepend semantics

		// +1 because the last line may lack a trailing \n.
		if bytes.Count(buf, []byte{'\n'}) >= n+1 {
			break
		}
	}

	scanner := bufio.NewScanner(bytes.NewReader(buf))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		cp := make([]byte, len(line))
		copy(cp, line)
		lines = append(lines, cp)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}

	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, nil
}

// LiveTail starts at EOF and reopens the file after rotation.
func LiveTail(ctx context.Context, path string) <-chan []byte {
	out := make(chan []byte, 64)

	go func() {
		defer close(out)

		var f *os.File
		for {
			var err error
			f, err = os.Open(path) //nolint:gosec // server-controlled config path
			if err == nil {
				break
			}
			if !errors.Is(err, fs.ErrNotExist) {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(500 * time.Millisecond):
			}
		}
		defer func() { _ = f.Close() }()

		if _, err := f.Seek(0, io.SeekEnd); err != nil {
			return
		}

		reader := bufio.NewReader(f)
		var carry []byte

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			line, err := reader.ReadBytes('\n')
			if len(line) > 0 {
				if line[len(line)-1] == '\n' {
					full := append(carry, line[:len(line)-1]...) //nolint:gocritic
					carry = nil
					select {
					case out <- full:
					case <-ctx.Done():
						return
					}
				} else {
					carry = append(carry, line...)
				}
				continue
			}

			if errors.Is(err, io.EOF) {
				if rotated, err := rotated(path, f); err == nil && rotated {
					_ = f.Close()
					var openErr error
					f, openErr = os.Open(path) //nolint:gosec
					if openErr != nil {
						return
					}
					reader = bufio.NewReader(f)
					continue
				}
				select {
				case <-ctx.Done():
					return
				case <-time.After(200 * time.Millisecond):
				}
				continue
			}
			if err != nil {
				return
			}
		}
	}()

	return out
}

func rotated(path string, open *os.File) (bool, error) {
	openStat, err := open.Stat()
	if err != nil {
		return false, err
	}
	pathStat, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !os.SameFile(openStat, pathStat) {
		return true, nil
	}
	pos, err := open.Seek(0, io.SeekCurrent)
	if err != nil {
		return false, err
	}
	if pathStat.Size() < pos {
		return true, nil
	}
	return false, nil
}
