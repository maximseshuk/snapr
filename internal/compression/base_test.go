package compression

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFactoryAliases(t *testing.T) {
	f := NewFactory()
	cases := []struct{ in, wantType string }{
		{"", "tar"},
		{"tar", "tar"},
		{"tar.gz", "tar.gz"},
		{"gz", "tar.gz"},
		{"gzip", "tar.gz"},
		{"tar.zst", "tar.zst"},
		{"zst", "tar.zst"},
		{"zstd", "tar.zst"},
		{"tar.xz", "tar.xz"},
		{"xz", "tar.xz"},
		{"zip", "zip"},
	}
	for _, tc := range cases {
		c := f.Create(tc.in)
		if assert.NotNilf(t, c, "Create(%q) should not be nil", tc.in) {
			assert.Equalf(t, tc.wantType, c.GetType(), "Create(%q)", tc.in)
		}
	}

	assert.Nil(t, f.Create("rar"), "unknown format must return nil")
}
