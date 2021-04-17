package registry

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/erikvanbrakel/anthology/app"
	"github.com/erikvanbrakel/anthology/models"
	"github.com/sirupsen/logrus"
)

type FilesystemRegistry struct {
	basePath string
}

func (r *FilesystemRegistry) ListModules(ctx context.Context, namespace, name, provider string, offset, limit int) (modules []models.Module, total int, err error) {

	modules, err = r.getModules(namespace, name, provider)

	count := len(modules)

	if err != nil {
		return nil, 0, err
	}

	if count == 0 {
		return modules[0:0], 0, nil
	}

	end := limit + offset
	if (end) > len(modules) {
		end = len(modules)
	}

	return modules[offset:end], len(modules), nil

}

func (r *FilesystemRegistry) PublishModule(ctx context.Context, namespace, name, provider, version string, data io.Reader) (err error) {
	panic("implement me")
}

func (r *FilesystemRegistry) GetModuleData(ctx context.Context, namespace, name, provider, version string) (reader *bytes.Buffer, err error) {
	panic("implement me")
}

func NewFilesystemRegistry(options app.FileSystemOptions) Registry {

	registry := FilesystemRegistry{basePath: options.BasePath}

	if !strings.HasSuffix(registry.basePath, string(os.PathSeparator)) {
		registry.basePath = registry.basePath + string(os.PathSeparator)
	}

	logrus.Infof("Using Filesystem Registry with basepath %s", registry.basePath)

	return &registry
}

func (r *FilesystemRegistry) getModules(namespace, name, provider string) ([]models.Module, error) {

	glob := r.basePath

	if namespace != "" {
		glob = path.Join(glob, namespace)
	} else {
		glob = path.Join(glob, "*")
	}

	if name != "" {
		glob = path.Join(glob, name)
	} else {
		glob = path.Join(glob, "*")
	}

	if provider != "" {
		glob = path.Join(glob, provider)
	} else {
		glob = path.Join(glob, "*")
	}

	glob = path.Join(glob, "*.tgz")

	var modules []models.Module

	dirs, err := filepath.Glob(glob)

	if err != nil {
		return nil, errors.New("unable to read module directories")
	}

	for _, f := range dirs {
		parts := strings.Split(strings.TrimPrefix(f, r.basePath), string(os.PathSeparator))

		if len(parts) != 4 {
			continue
		}

		modules = append(modules, models.Module{
			Namespace: parts[0],
			Name:      parts[1],
			Provider:  parts[2],
			Version:   strings.TrimRight(parts[3], ".tgz"),
		})
	}

	return modules, nil
}
