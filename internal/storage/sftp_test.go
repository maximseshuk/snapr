package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"

	pkgconfig "github.com/maximseshuk/snapr/internal/config"
)

func TestSFTP_JobDir(t *testing.T) {
	s := NewSFTPStorage()
	cases := []struct {
		name    string
		storage pkgconfig.StorageConfig
		job     string
		want    string
	}{
		{"empty path defaults to .", pkgconfig.StorageConfig{}, "myjob", "myjob"},
		{"absolute base", pkgconfig.StorageConfig{Path: "/srv/backups"}, "myjob", "/srv/backups/myjob"},
		{"relative base", pkgconfig.StorageConfig{Path: "backups"}, "myjob", "backups/myjob"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, s.jobDir(c.storage, c.job))
		})
	}
}
