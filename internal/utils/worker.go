package utils

import "strconv"

func WorkerCount(extra map[string]string, def, max int) int {
	if v, ok := extra["workers"]; ok {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= max {
			return n
		}
	}
	return def
}
