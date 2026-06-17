package storage

import (
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

const splitWrapperMarker = ".parts-"

// jobNameSegment returns the job-name path segment to append to a storage's
// base path, honouring storage.IncludeJobName (nil/true → append, false → skip).
func jobNameSegment(includeJobName *bool, jobName string) string {
	if includeJobName != nil && !*includeJobName {
		return ""
	}
	return jobName
}

func JobDirPosix(basePath, jobName string) string {
	if basePath == "" {
		return jobName
	}
	return path.Join(basePath, jobName)
}

func JobDirLocal(basePath, jobName string) string {
	return filepath.Join(basePath, jobName)
}

func SplitWrapperName(archiveName string, partsCount int, totalSize int64) string {
	return fmt.Sprintf("%s%s%d-%d", archiveName, splitWrapperMarker, partsCount, totalSize)
}

func ParseSplitWrapper(name string) (base string, parts int, totalSize int64, ok bool) {
	idx := strings.LastIndex(name, splitWrapperMarker)
	if idx == -1 {
		return "", 0, 0, false
	}
	suffix := name[idx+len(splitWrapperMarker):]
	dash := strings.IndexByte(suffix, '-')
	if dash <= 0 {
		return "", 0, 0, false
	}
	n, err := strconv.Atoi(suffix[:dash])
	if err != nil || n <= 0 {
		return "", 0, 0, false
	}
	sz, err := strconv.ParseInt(suffix[dash+1:], 10, 64)
	if err != nil || sz < 0 {
		return "", 0, 0, false
	}
	return name[:idx], n, sz, true
}
