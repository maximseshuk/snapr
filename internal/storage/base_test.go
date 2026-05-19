package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFactory_Create(t *testing.T) {
	f := NewFactory()
	for _, name := range []string{"s3", "local", "bunny", "sftp", "webdav"} {
		t.Run(name, func(t *testing.T) {
			s := f.Create(name)
			require.NotNil(t, s)
			assert.Equal(t, name, s.GetType())
		})
	}
}

func TestFactory_UnknownReturnsNil(t *testing.T) {
	assert.Nil(t, NewFactory().Create("ftp"))
}
