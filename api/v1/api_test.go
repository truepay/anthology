package v1_test

import (
	"context"
	"net/http/httptest"
	"testing"

	"bytes"
	"net/http"

	v1 "github.com/erikvanbrakel/anthology/api/v1"
	"github.com/erikvanbrakel/anthology/app"
	"github.com/erikvanbrakel/anthology/registry"
	"github.com/erikvanbrakel/anthology/services"
	"github.com/gavv/httpexpect"
	routing "github.com/go-ozzo/ozzo-routing"
	"github.com/go-ozzo/ozzo-routing/content"
	"github.com/sirupsen/logrus"
)

type apiTestCase struct {
	tag    string
	method string
	url    string
	body   string
	status int
	assert func(*testing.T, *httpexpect.Response, *httptest.Server)
}

func newRouter() *routing.Router {
	logger := logrus.New()
	logger.Level = logrus.PanicLevel

	router := routing.New()

	router.Use(
		app.Init(logger),
		content.TypeNegotiator(content.JSON),
	)

	return router
}

func runAPITests(t *testing.T, dataset []testModule, tests []apiTestCase) {
	for _, test := range tests {
		t.Run(test.tag, func(t *testing.T) {
			ctx := context.Background()
			r := registry.NewFakeRegistry()
			defer r.Close()

			for _, m := range dataset {
				r.PublishModule(ctx, m.namespace, m.name, m.provider, m.version, bytes.NewBuffer(m.data))
			}

			router := newRouter()
			v1.ServeModuleResource(&router.RouteGroup, services.NewModuleService(r))
			server := httptest.NewServer(router)
			defer server.Close()

			e := httpexpect.New(t, server.URL)

			result := e.Request(test.method, test.url).
				WithClient(&http.Client{
					CheckRedirect: func(req *http.Request, via []*http.Request) error {
						return http.ErrUseLastResponse
					},
				}).
				WithHeader("Content-Type", "application/json").
				WithBytes([]byte(test.body)).
				Expect().Status(test.status)

			test.assert(t, result, server)
		})
	}
}

type testModule struct {
	namespace string
	name      string
	provider  string
	version   string
	data      []byte
}
