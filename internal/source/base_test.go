package source

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFactory_Create(t *testing.T) {
	f := NewFactory()

	cases := []struct {
		typeName string
		want     string
	}{
		{"postgresql", "postgresql"},
		{"mysql", "mysql"},
		{"mariadb", "mariadb"},
		{"mongodb", "mongodb"},
		{"redis", "redis"},
		{"sqlite", "sqlite"},
		{"bunny", "bunny"},
		{"local", "local"},
		{"s3", "s3"},
	}
	for _, c := range cases {
		t.Run(c.typeName, func(t *testing.T) {
			s := f.Create(c.typeName)
			require.NotNil(t, s, "factory must return non-nil for %s", c.typeName)
			assert.Equal(t, c.want, s.GetType())
		})
	}
}

func TestFactory_UnknownReturnsNil(t *testing.T) {
	assert.Nil(t, NewFactory().Create("does-not-exist"))
}
