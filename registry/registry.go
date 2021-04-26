package registry

import (
	"bytes"
	"context"
	"io"

	"github.com/erikvanbrakel/anthology/models"
)

type Registry interface {
	GetModuleData(ctx context.Context, namespace, name, provider, version string) (reader *bytes.Buffer, err error)
	ListModules(ctx context.Context, namespace, name, provider string, offset, limit int) (modules []models.Module, total int, err error)
	PublishModule(ctx context.Context, namespace, name, provider, version string, data io.Reader) (err error)
	Close()
}
