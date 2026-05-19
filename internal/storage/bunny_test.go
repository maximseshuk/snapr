package storage

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	pkgconfig "github.com/maximseshuk/snapr/internal/config"
)

func bunnyCfg() pkgconfig.StorageConfig {
	return pkgconfig.StorageConfig{
		Endpoint:  "https://storage.bunnycdn.com",
		ZoneName:  "myzone",
		AccessKey: "k",
	}
}

func TestBunny_BaseURL(t *testing.T) {
	b := NewBunnyStorage()
	assert.Equal(t, "https://storage.bunnycdn.com/myzone", b.baseURL(bunnyCfg()))

	cfg := bunnyCfg()
	cfg.Endpoint = "https://storage.bunnycdn.com/"
	assert.Equal(t, "https://storage.bunnycdn.com/myzone", b.baseURL(cfg))
}

func TestBunny_JobBaseURL(t *testing.T) {
	b := NewBunnyStorage()
	cases := []struct {
		name string
		cfg  pkgconfig.StorageConfig
		want string
	}{
		{"no path", bunnyCfg(), "https://storage.bunnycdn.com/myzone/myjob"},
		{"with path", with(bunnyCfg(), func(c *pkgconfig.StorageConfig) { c.Path = "backups" }), "https://storage.bunnycdn.com/myzone/backups/myjob"},
		{"path with slashes", with(bunnyCfg(), func(c *pkgconfig.StorageConfig) { c.Path = "/a/b/" }), "https://storage.bunnycdn.com/myzone/a/b/myjob"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, b.jobBaseURL(c.cfg, "myjob"))
		})
	}
}

func TestBunny_ObjectURL(t *testing.T) {
	b := NewBunnyStorage()
	assert.Equal(t,
		"https://storage.bunnycdn.com/myzone/myjob/file.tar.gz",
		b.objectURL(bunnyCfg(), "myjob", "", "file.tar.gz"))

	assert.Equal(t,
		"https://storage.bunnycdn.com/myzone/myjob/wrap.parts-2-100/file.tar.gz",
		b.objectURL(bunnyCfg(), "myjob", "wrap.parts-2-100", "file.tar.gz"))
}

func TestBunny_ObjectURL_EscapesFilename(t *testing.T) {
	b := NewBunnyStorage()
	got := b.objectURL(bunnyCfg(), "myjob", "", "name with space.tar.gz")
	assert.Contains(t, got, "name%20with%20space.tar.gz")
}

func TestBunny_ListURL(t *testing.T) {
	b := NewBunnyStorage()
	assert.Equal(t,
		"https://storage.bunnycdn.com/myzone/myjob/",
		b.listURL(bunnyCfg(), "myjob", ""))
	assert.Equal(t,
		"https://storage.bunnycdn.com/myzone/myjob/wrap/",
		b.listURL(bunnyCfg(), "myjob", "wrap"))
}

func TestBunny_SignPullZoneURL(t *testing.T) {
	b := NewBunnyStorage()
	cfg := bunnyCfg()
	cfg.PullZoneHostname = "myzone.b-cdn.net"
	cfg.PullZoneTokenAuthKey = "secret"
	cfg.PullZoneTokenTTL = 60

	t.Run("no path, no wrapper", func(t *testing.T) {
		got := b.signPullZoneURL(cfg, "myjob", "", "f.tar.gz")
		assert.True(t, strings.HasPrefix(got, "https://myzone.b-cdn.net/myjob/f.tar.gz?"), "got %q", got)
		assert.Contains(t, got, "token=HS256-")
	})

	t.Run("with path and wrapper", func(t *testing.T) {
		c := cfg
		c.Path = "/backups/"
		got := b.signPullZoneURL(c, "myjob", "wrap", "f.tar.gz")
		assert.True(t, strings.HasPrefix(got, "https://myzone.b-cdn.net/backups/myjob/wrap/f.tar.gz?"), "got %q", got)
	})
}

func with(c pkgconfig.StorageConfig, fn func(*pkgconfig.StorageConfig)) pkgconfig.StorageConfig {
	fn(&c)
	return c
}
