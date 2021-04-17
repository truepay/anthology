package v1

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/blang/semver"
	"github.com/erikvanbrakel/anthology/app"
	"github.com/erikvanbrakel/anthology/models"
	routing "github.com/go-ozzo/ozzo-routing"
)

type (
	moduleService interface {
		Query(ctx context.Context, rs app.RequestScope, namespace, name, provider string, verified bool, offset, limit int) ([]models.Module, int, error)
		QueryVersions(ctx context.Context, rs app.RequestScope, namespace, name, provider string) ([]models.Module, error)
		Exists(ctx context.Context, rs app.RequestScope, namespace, name, provider, version string) (bool, error)
		Get(ctx context.Context, rs app.RequestScope, namespace, name, provider, version string) (*models.Module, error)
		GetData(ctx context.Context, rs app.RequestScope, namespace, name, provider, version string) (io.Reader, error)
		Publish(ctx context.Context, rs app.RequestScope, namespace, name, provider, version string, data io.Reader) error
	}

	moduleResource struct {
		service moduleService
	}

	apiError struct {
		Errors []string `json:"errors"`
	}
)

func ServeModuleResource(rg *routing.RouteGroup, service moduleService) {
	r := &moduleResource{service}

	// List modules
	rg.Get("/", r.query)
	rg.Get("/<namespace>", r.query)

	// Search modules
	rg.Get("/search")

	// List available versions for a specific module
	rg.Get("/<namespace>/<name>/<provider>/versions", r.queryVersions)

	// Download source code for a specific module version
	rg.Get("/<namespace>/<name>/<provider>/<version>/download", r.getDownloadUrl).Name("GetDownloadUrl")

	// Download the latest version of a module
	rg.Get("/<namespace>/<name>/<provider>/download", r.getLatestDownloadUrl)

	// List latest version of module for all providers
	rg.Get("/<namespace>/<name>", r.queryLatest)

	// Latest version for a specific module provider
	rg.Get("/<namespace>/<name>/<provider>", r.getLatest)

	// Get a specific module
	rg.Get("/<namespace>/<name>/<provider>/<version>", r.get)

	// Publish a specific module
	rg.Post("/<namespace>/<name>/<provider>/<version>", r.publish)

	rg.Get("/<namespace>/<name>/<provider>/<version>/data.tgz", r.getModuleData).Name("GetModuleData")
}

func (r *moduleResource) getModuleData(c *routing.Context) error {
	rs := app.GetRequestScope(c)

	namespace := c.Param("namespace")
	name := c.Param("name")
	provider := c.Param("provider")
	version := c.Param("version")

	data, err := r.service.GetData(c.Request.Context(), rs, namespace, name, provider, version)

	if err != nil {
		return err
	}

	_, err = io.Copy(c.Response, data)
	return err
}

func (r *moduleResource) publish(c *routing.Context) error {
	rs := app.GetRequestScope(c)
	namespace, name, provider, version := c.Param("namespace"), c.Param("name"), c.Param("provider"), c.Param("version")

	err := r.service.Publish(c.Request.Context(), rs, namespace, name, provider, version, c.Request.Body)
	if err != nil {
		return err
	}

	return r.getDownloadUrl(c)
}

func (r *moduleResource) query(c *routing.Context) error {
	rs := app.GetRequestScope(c)

	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	namespace := c.Param("namespace")
	provider := c.Query("provider", "")
	verified, _ := strconv.ParseBool(c.Query("verified", "false"))

	modules, count, err := r.service.Query(c.Request.Context(), rs, namespace, "", provider, verified, offset, limit)

	if err != nil {
		return err
	}

	if count == 0 {
		c.Response.WriteHeader(http.StatusNotFound)
		return c.Write(apiError{[]string{"not found"}})
	}

	paginationInfo := getPaginationInfo(c, count)

	return c.Write(PaginatedList{
		PaginationInfo: paginationInfo,
		Modules:        modules,
	})
}

func (r *moduleResource) queryVersions(c *routing.Context) error {
	rs := app.GetRequestScope(c)

	namespace := c.Param("namespace")
	name := c.Param("name")
	provider := c.Param("provider")

	versionsByModule, err := r.service.QueryVersions(c.Request.Context(), rs, namespace, name, provider)

	if err != nil {
		return err
	}

	if len(versionsByModule) == 0 {
		c.Response.WriteHeader(http.StatusNotFound)
		return c.Write(apiError{[]string{"not found"}})
	}

	return c.Write(struct {
		Modules VersionsList `json:"modules"`
	}{
		VersionsList{

			{
				Source:   fmt.Sprintf("%s/%s/%s", namespace, name, provider),
				Versions: versionsByModule,
			},
		},
	})
}

func (r *moduleResource) getDownloadUrl(c *routing.Context) error {
	namespace, name, provider, version := c.Param("namespace"), c.Param("name"), c.Param("provider"), c.Param("version")

	if exists, _ := r.service.Exists(c.Request.Context(), app.GetRequestScope(c), namespace, name, provider, version); exists {

		url := c.URL("GetModuleData",
			"namespace", namespace,
			"name", name,
			"provider", provider,
			"version", version,
		)
		c.Response.Header().Set("X-Terraform-Get", url)
		c.Response.WriteHeader(http.StatusNoContent)
		return nil
	}

	c.Response.WriteHeader(http.StatusNotFound)
	return c.Write(apiError{[]string{"not found"}})
}

func (r *moduleResource) getLatestDownloadUrl(c *routing.Context) error {
	rs := app.GetRequestScope(c)

	namespace := c.Param("namespace")
	name := c.Param("name")
	provider := c.Param("provider")

	modules, count, err := r.service.Query(c.Request.Context(), rs, namespace, name, provider, false, 0, 100000)

	if err != nil {
		return err
	}

	if count == 0 {
		c.Response.WriteHeader(http.StatusNotFound)
		return c.Write(apiError{[]string{"not found"}})
	}

	var latestVersion semver.Version

	for _, m := range modules {
		moduleVersion, _ := semver.Make(m.Version)
		if moduleVersion.Compare(latestVersion) > 0 {
			latestVersion = moduleVersion
		}
	}

	url := c.URL("GetDownloadUrl",
		"namespace", c.Param("namespace"),
		"name", c.Param("name"),
		"provider", c.Param("provider"),
		"version", latestVersion.String(),
	)

	c.Response.Header().Set("Location", url)
	c.Response.WriteHeader(http.StatusFound)
	return nil
}

func (r *moduleResource) get(c *routing.Context) error {
	rs := app.GetRequestScope(c)

	namespace := c.Param("namespace")
	name := c.Param("name")
	provider := c.Param("provider")
	version := c.Param("version")

	module, err := r.service.Get(c.Request.Context(), rs, namespace, name, provider, version)

	if err != nil {
		return err
	}

	if module == nil {
		c.Response.WriteHeader(http.StatusNotFound)
		return c.Write(apiError{[]string{"not found"}})
	}

	return c.Write(module)
}

func (r *moduleResource) getLatest(c *routing.Context) error {
	rs := app.GetRequestScope(c)

	namespace := c.Param("namespace")
	name := c.Param("name")
	provider := c.Param("provider")

	modules, err := r.service.QueryVersions(c.Request.Context(), rs, namespace, name, provider)

	if err != nil {
		return err
	}

	if len(modules) == 0 {
		c.Response.WriteHeader(http.StatusNotFound)
		return c.Write(apiError{[]string{"not found"}})
	}

	var module = models.Module{
		Version: "0.0.0",
	}

	for _, m := range modules {
		moduleVersion, _ := semver.Make(m.Version)
		latestVersion, _ := semver.Make(module.Version)

		if moduleVersion.Compare(latestVersion) > 0 {
			module = m
		}
	}

	return c.Write(module)
}

func (r *moduleResource) queryLatest(c *routing.Context) error {
	rs := app.GetRequestScope(c)

	namespace := c.Param("namespace")
	name := c.Param("name")

	modules, err := r.service.QueryVersions(c.Request.Context(), rs, namespace, name, "")

	if err != nil {
		return err
	}

	if len(modules) == 0 {
		c.Response.WriteHeader(http.StatusNotFound)
		return c.Write(apiError{[]string{"not found"}})
	}

	var latestVersions = map[string]models.Module{}

	for _, m := range modules {
		latestVersion := latestVersions[m.Provider]
		latest, _ := semver.Make(latestVersion.Version)
		current, _ := semver.Make(m.Version)

		if current.Compare(latest) > 0 {
			latestVersions[m.Provider] = m
		}
	}

	v := make([]models.Module, 0, len(latestVersions))

	for _, value := range latestVersions {
		v = append(v, value)
	}

	return c.Write(PaginatedList{
		PaginationInfo: getPaginationInfo(c, len(v)),
		Modules:        v,
	})
}

type VersionsList []struct {
	Source   string          `json:"source"`
	Versions []models.Module `json:"versions"`
}

func getPaginationInfo(c *routing.Context, count int) PaginationInfo {
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	return PaginationInfo{
		CurrentOffset: offset,
		Limit:         limit,
	}
}

type PaginatedList struct {
	PaginationInfo PaginationInfo `json:"meta"`
	Modules        interface{}    `json:"modules"`
}

type PaginationInfo struct {
	Limit          int    `json:"limit"`
	PreviousOffset int    `json:"previous_offset"`
	PreviousUrl    int    `json:"previous_url"`
	CurrentOffset  int    `json:"current_offset"`
	NextOffset     int    `json:"next_offset"`
	NextUrl        string `json:"next_url"`
}
