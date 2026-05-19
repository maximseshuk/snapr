package encryptor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maximseshuk/snapr/internal/config"
)

func TestOpenSSLGetType(t *testing.T) {
	assert.Equal(t, "openssl", NewOpenSSL().GetType())
}

func TestOpenSSLEncryptRequiresPassword(t *testing.T) {
	_, err := NewOpenSSL().Encrypt(context.Background(), "/tmp/whatever", config.EncryptionConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "password")
}

func TestOpenSSLEncryptDecryptRoundTrip(t *testing.T) {
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl not on PATH")
	}

	dir := t.TempDir()
	archivePath := filepath.Join(dir, "data.tar")
	payload := []byte("snapr round-trip payload\n")
	require.NoError(t, os.WriteFile(archivePath, payload, 0o600))

	cfg := config.EncryptionConfig{
		Type:     "openssl",
		Cipher:   "aes-256-cbc",
		Password: "correct-horse-battery-staple",
	}

	encPath, err := NewOpenSSL().Encrypt(context.Background(), archivePath, cfg)
	require.NoError(t, err)
	assert.Equal(t, archivePath+".enc", encPath)

	_, err = os.Stat(archivePath)
	assert.True(t, os.IsNotExist(err), "plaintext archive must be deleted")

	encBytes, err := os.ReadFile(encPath)
	require.NoError(t, err)
	assert.NotEqual(t, payload, encBytes, "ciphertext must differ from plaintext")

	decPath := encPath + ".dec"
	cmd := exec.Command("openssl", "enc", "-d", "-aes-256-cbc", "-pbkdf2",
		"-in", encPath, "-out", decPath, "-pass", "env:SNAPR_TEST_PASS")
	cmd.Env = append(os.Environ(), "SNAPR_TEST_PASS="+cfg.Password)

	var stderr strings.Builder
	cmd.Stderr = &stderr
	require.NoErrorf(t, cmd.Run(), "openssl decrypt: %s", stderr.String())

	got, err := os.ReadFile(decPath)
	require.NoError(t, err)
	assert.Equal(t, payload, got, "decrypted bytes must match the original")
}

func TestOpenSSLDecryptWrongPassword(t *testing.T) {
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl not on PATH")
	}

	dir := t.TempDir()
	archivePath := filepath.Join(dir, "data.tar")
	require.NoError(t, os.WriteFile(archivePath, []byte("payload"), 0o600))

	encPath, err := NewOpenSSL().Encrypt(context.Background(), archivePath, config.EncryptionConfig{
		Password: "right-password",
	})
	require.NoError(t, err)

	cmd := exec.Command("openssl", "enc", "-d", "-aes-256-cbc", "-pbkdf2",
		"-in", encPath, "-out", filepath.Join(dir, "out"), "-pass", "env:WRONG")
	cmd.Env = append(os.Environ(), "WRONG=nope")
	assert.Error(t, cmd.Run(), "decrypt with wrong password must fail")
}

func TestOpenSSLDefaultCipher(t *testing.T) {
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("openssl not on PATH")
	}

	dir := t.TempDir()
	archivePath := filepath.Join(dir, "data.tar")
	require.NoError(t, os.WriteFile(archivePath, []byte("hello"), 0o600))

	encPath, err := NewOpenSSL().Encrypt(context.Background(), archivePath, config.EncryptionConfig{
		Password: "pw",
	})
	require.NoError(t, err)

	cmd := exec.Command("openssl", "enc", "-d", "-aes-256-cbc", "-pbkdf2",
		"-in", encPath, "-out", filepath.Join(dir, "out"), "-pass", "env:P")
	cmd.Env = append(os.Environ(), "P=pw")
	require.NoError(t, cmd.Run())
}
