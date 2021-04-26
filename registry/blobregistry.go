package registry

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"

	"github.com/erikvanbrakel/anthology/models"
	"gocloud.dev/blob"
)

const (
	moduleExtension = ".tgz"
)

type blobRegistry struct {
	bucket *blob.Bucket
}

func (r *blobRegistry) ListModules(ctx context.Context, namespace, name, provider string, offset, limit int) ([]models.Module, int, error) {
	modules := []models.Module{}

	prefix := r.modulePrefix(namespace, name, provider)
	iter := r.bucket.List(&blob.ListOptions{Prefix: prefix})
	for {
		obj, err := iter.Next(ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, err
		}
		if obj.IsDir {
			continue
		}

		parts := strings.Split(obj.Key, "/")

		if len(parts) == 4 {
			if namespace != "" && parts[0] != namespace {
				continue
			}
			if name != "" && parts[1] != name {
				continue
			}
			if provider != "" && parts[2] != provider {
				continue
			}

			modules = append(modules, models.Module{
				Namespace: parts[0],
				Name:      parts[1],
				Provider:  parts[2],
				Version:   strings.TrimRight(parts[3], moduleExtension),
			})
		}
	}

	if offset >= len(modules) || len(modules) == 0 || limit == 0 {
		return nil, len(modules), nil
	}
	low := offset
	high := offset + limit
	if high >= len(modules) {
		high = len(modules)
	}

	return modules[low:high], len(modules), nil
}

func (r *blobRegistry) PublishModule(ctx context.Context, namespace, name, provider, version string, data io.Reader) error {
	key := r.moduleKey(namespace, name, provider, version)
	writer, err := r.bucket.NewWriter(ctx, key, nil)
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = io.Copy(writer, data)
	if err != nil {
		return err
	}

	return nil
}

func (r *blobRegistry) GetModuleData(ctx context.Context, namespace, name, provider, version string) (*bytes.Buffer, error) {
	key := r.moduleKey(namespace, name, provider, version)

	exists, err := r.bucket.Exists(ctx, key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("module does not exist")
	}

	data, err := r.bucket.ReadAll(ctx, key)
	if err != nil {
		return nil, err
	}

	buffer := bytes.NewBuffer(data)
	return buffer, nil
}

func (r *blobRegistry) Close() {
	r.bucket.Close()
}

func (r *blobRegistry) modulePrefix(namespace, name, provider string) string {
	parts := make([]string, 0, 3)
	if namespace != "" {
		parts = append(parts, namespace)
		if name != "" {
			parts = append(parts, name)
			if provider != "" {
				parts = append(parts, provider)
			}
		}
	}
	return strings.Join(parts, "/")
}

func (r *blobRegistry) moduleKey(namespace, name, provider, version string) string {
	return strings.Join([]string{namespace, name, provider, version}, "/") + moduleExtension
}

func newBlobRegistry(bucket *blob.Bucket) Registry {
	return &blobRegistry{
		bucket,
	}
}
