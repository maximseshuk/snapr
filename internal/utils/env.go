package utils

import "os"

func GetEnvironment() string {
	if env := os.Getenv("ENV"); env != "" {
		return env
	}
	return "development"
}
