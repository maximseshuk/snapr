package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"

	pkgconfig "github.com/maximseshuk/snapr/internal/config"
)

func TestWebdavDir(t *testing.T) {
	cases := map[string]string{
		"":         "/",
		"/":        "/",
		"backups":  "/backups",
		"/backups": "/backups",
		"a/b":      "/a/b",
	}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			assert.Equal(t, want, webdavDir(pkgconfig.StorageConfig{Path: in}))
		})
	}
}

func TestWebDAV_JobDir(t *testing.T) {
	w := NewWebDAVStorage()
	cases := []struct {
		name    string
		storage pkgconfig.StorageConfig
		job     string
		want    string
	}{
		{"root", pkgconfig.StorageConfig{}, "myjob", "/myjob"},
		{"with path", pkgconfig.StorageConfig{Path: "backups"}, "myjob", "/backups/myjob"},
		{"absolute path", pkgconfig.StorageConfig{Path: "/srv"}, "myjob", "/srv/myjob"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, w.jobDir(c.storage, c.job))
		})
	}
}
