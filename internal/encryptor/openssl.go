package encryptor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rs/zerolog"

	"github.com/maximseshuk/snapr/internal/config"
	"github.com/maximseshuk/snapr/internal/utils"
)

type OpenSSL struct{}

func NewOpenSSL() *OpenSSL { return &OpenSSL{} }

func (o *OpenSSL) GetType() string { return "openssl" }

func (o *OpenSSL) Encrypt(ctx context.Context, archivePath string, cfg config.EncryptionConfig) (string, error) {
	logger := zerolog.Ctx(ctx)

	if cfg.Password == "" {
		return "", fmt.Errorf("encryption password is required")
	}

	cipher := cfg.Cipher
	if cipher == "" {
		cipher = "aes-256-cbc"
	}

	encryptedPath := archivePath + ".enc"

	args := []string{
		"enc", "-" + cipher,
		"-salt",
		"-pbkdf2",
		"-in", archivePath,
		"-out", encryptedPath,
		"-pass", "env:SNAPR_ENC_PASS",
	}

	cmd := exec.CommandContext(ctx, "openssl", args...)
	cmd.Env = append(os.Environ(), "SNAPR_ENC_PASS="+cfg.Password)

	var stderr strings.Builder
	cmd.Stderr = &stderr

	logger.Info().Str("cipher", cipher).Str("output", encryptedPath).Msg("Encrypting archive")

	if err := cmd.Run(); err != nil {
		_ = os.Remove(encryptedPath)
		return "", fmt.Errorf("openssl encrypt failed: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}

	if err := os.Remove(archivePath); err != nil {
		logger.Warn().Err(err).Str("path", archivePath).Msg("Could not remove plaintext archive")
	}

	if fi, err := os.Stat(encryptedPath); err == nil {
		logger.Info().
			Str("path", encryptedPath).
			Int64("size_bytes", fi.Size()).
			Str("size_human", utils.FormatBytes(fi.Size())).
			Msg("Archive encrypted")
	}

	return encryptedPath, nil
}
