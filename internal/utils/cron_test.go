package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetNextRunTime(t *testing.T) {
	now := time.Now()
	next, err := GetNextRunTime("* * * * *")
	require.NoError(t, err)
	assert.True(t, next.After(now), "next must be in the future")
	assert.True(t, next.Sub(now) <= time.Minute+time.Second, "next within ~1 minute")
}

func TestGetNextRunTime_Invalid(t *testing.T) {
	_, err := GetNextRunTime("not a cron")
	assert.Error(t, err)
}
