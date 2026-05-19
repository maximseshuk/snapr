package compression

import (
	"context"
)

type Compressor interface {
	Compress(ctx context.Context, sourcesDir, tmpDir, archiveName string) (string, error)
	GetType() string
	GetExtension() string
}

type Factory struct {
	compressors map[string]Compressor
}

func NewFactory() *Factory {
	factory := &Factory{
		compressors: make(map[string]Compressor),
	}

	factory.Register(NewTarCompressor())
	factory.Register(NewTarGzCompressor())
	factory.Register(NewTarZstCompressor())
	factory.Register(NewTarXzCompressor())
	factory.Register(NewZipCompressor())

	return factory
}

func (f *Factory) Register(compressor Compressor) {
	f.compressors[compressor.GetType()] = compressor
}

func (f *Factory) Create(compressionType string) Compressor {
	switch compressionType {
	case "", "tar":
		compressionType = "tar"
	case "gz", "gzip", "tar.gz":
		compressionType = "tar.gz"
	case "zst", "zstd", "tar.zst":
		compressionType = "tar.zst"
	case "xz", "tar.xz":
		compressionType = "tar.xz"
	}

	return f.compressors[compressionType]
}
