package encryptor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFactoryCreate(t *testing.T) {
	f := NewFactory()

	cases := []struct{ in, wantType string }{
		{"", "openssl"},
		{"openssl", "openssl"},
	}
	for _, tc := range cases {
		enc := f.Create(tc.in)
		require.NotNilf(t, enc, "Create(%q)", tc.in)
		assert.Equal(t, tc.wantType, enc.GetType())
	}

	assert.Nil(t, f.Create("unknown"), "unknown type must return nil")
}
