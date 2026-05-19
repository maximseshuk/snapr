package source

import (
	"context"

	"github.com/maximseshuk/snapr/internal/config"
)

type Source interface {
	Backup(ctx context.Context, destDir string, source config.SourceConfig) error
	GetType() string
}

type Factory struct {
	sources map[string]Source
}

func NewFactory() *Factory {
	factory := &Factory{
		sources: make(map[string]Source),
	}

	factory.Register(NewPostgreSQLSource())
	factory.Register(NewMySQLSource())
	factory.Register(NewMariaDBSource())
	factory.Register(NewMongoDBSource())
	factory.Register(NewRedisSource())
	factory.Register(NewSQLiteSource())
	factory.Register(NewBunnySource())
	factory.Register(NewLocalSource())
	factory.Register(NewS3Source())

	return factory
}

func (f *Factory) Register(source Source) {
	f.sources[source.GetType()] = source
}

func (f *Factory) Create(sourceType string) Source {
	return f.sources[sourceType]
}
