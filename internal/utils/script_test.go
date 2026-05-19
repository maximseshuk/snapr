package utils

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecScript_Empty(t *testing.T) {
	assert.NoError(t, ExecScript(context.Background(), "", nil))
	assert.NoError(t, ExecScript(context.Background(), "   \n\t  ", nil))
}

func TestExecScript_Stdout(t *testing.T) {
	var buf bytes.Buffer
	err := ExecScript(context.Background(), "echo hello", &buf)
	require.NoError(t, err)
	assert.Equal(t, "hello", strings.TrimSpace(buf.String()))
}

func TestExecScript_Stderr(t *testing.T) {
	var buf bytes.Buffer
	err := ExecScript(context.Background(), "echo oops 1>&2", &buf)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "oops")
}

func TestExecScript_Failure(t *testing.T) {
	err := ExecScript(context.Background(), "exit 3", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "script failed")
}

func TestExecScript_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := ExecScript(ctx, "sleep 5", nil)
	assert.Error(t, err)
}

func TestExecScript_NilWriter(t *testing.T) {
	assert.NoError(t, ExecScript(context.Background(), "echo discarded", nil))
}
