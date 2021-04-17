package services

import (
	"context"
	"io"

	"github.com/erikvanbrakel/anthology/app"
	"github.com/erikvanbrakel/anthology/models"
	"github.com/erikvanbrakel/anthology/registry"
)

type ModuleService struct {
	Registry registry.Registry
}

func NewModuleService(r registry.Registry) *ModuleService {
	return &ModuleService{
		r,
	}
}

func (s *ModuleService) Query(ctx context.Context, rs app.RequestScope, namespace, name, provider string, verified bool, offset, limit int) ([]models.Module, int, error) {

	modules, count, err := s.Registry.ListModules(ctx, namespace, name, provider, offset, limit)

	if err != nil {
		return nil, 0, err
	}

	return modules, count, nil
}

func (s *ModuleService) QueryVersions(ctx context.Context, rs app.RequestScope, namespace, name, provider string) ([]models.Module, error) {

	modules, _, err := s.Registry.ListModules(ctx, namespace, name, provider, 0, 10000)
	return modules, err
}

func (s *ModuleService) Exists(ctx context.Context, rs app.RequestScope, namespace, name, provider, version string) (bool, error) {
	modules, _, err := s.Registry.ListModules(ctx, namespace, name, provider, 0, 10000)

	if err != nil {
		return false, err
	}

	for _, m := range modules {
		if m.Version == version {
			return true, nil
		}
	}
	return false, nil
}

func (s *ModuleService) Get(ctx context.Context, rs app.RequestScope, namespace, name, provider, version string) (*models.Module, error) {
	modules, _, err := s.Registry.ListModules(ctx, namespace, name, provider, 0, 10000)

	if err != nil {
		return nil, err
	}

	for _, m := range modules {
		if m.Version == version {
			return &m, nil
		}
	}

	return nil, nil
}

func (s *ModuleService) Publish(ctx context.Context, rs app.RequestScope, namespace, name, provider, version string, data io.Reader) error {
	return s.Registry.PublishModule(ctx, namespace, name, provider, version, data)
}

func (s *ModuleService) GetData(ctx context.Context, rs app.RequestScope, namespace, name, provider, version string) (io.Reader, error) {
	return s.Registry.GetModuleData(ctx, namespace, name, provider, version)
}
