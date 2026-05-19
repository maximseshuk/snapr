package encryptor

import (
	"context"

	"github.com/maximseshuk/snapr/internal/config"
)

type Encryptor interface {
	Encrypt(ctx context.Context, archivePath string, cfg config.EncryptionConfig) (string, error)
	GetType() string
}

type Factory struct {
	encryptors map[string]Encryptor
}

func NewFactory() *Factory {
	factory := &Factory{
		encryptors: make(map[string]Encryptor),
	}

	factory.Register(NewOpenSSL())

	return factory
}

func (f *Factory) Register(encryptor Encryptor) {
	f.encryptors[encryptor.GetType()] = encryptor
}

func (f *Factory) Create(encryptionType string) Encryptor {
	if encryptionType == "" {
		encryptionType = "openssl"
	}
	return f.encryptors[encryptionType]
}
