package meta

import (
	"github.com/tomanagle/Social-Mockery/internal/providers/meta/generated/v24"
	"github.com/tomanagle/Social-Mockery/internal/providers/meta/generated/v25"
	"net/http"
)

// Each upstream version has its own generated interface and HTTP adapter.
// Shared simulation behavior is intentional for the currently implemented subset;
// adapters can diverge without changing another version's generated contract.
func versionRouters(p *Provider) map[string]http.Handler {
	return map[string]http.Handler{
		"v24.0": v24.HandlerWithOptions(&adapter24{p}, v24.StdHTTPServerOptions{ErrorHandlerFunc: parameterError}),
		"v25.0": v25.HandlerWithOptions(&adapter25{p}, v25.StdHTTPServerOptions{ErrorHandlerFunc: parameterError}),
	}
}
func parameterError(w http.ResponseWriter, r *http.Request, err error) {
	failure(w, 400, 100, "Invalid endpoint parameter")
}

type adapter24 struct{ provider *Provider }

var _ v24.ServerInterface = (*adapter24)(nil)

func (a *adapter24) GetMe(w http.ResponseWriter, r *http.Request, params v24.GetMeParams) {
	a.provider.operation(w, r, "me.read")
}
func (a *adapter24) GetObject(w http.ResponseWriter, r *http.Request, objectID string, params v24.GetObjectParams) {
	a.provider.operation(w, r, "object.read")
}
func (a *adapter24) ListMedia(w http.ResponseWriter, r *http.Request, objectID string, params v24.ListMediaParams) {
	a.provider.operation(w, r, "media.list")
}
func (a *adapter24) CreateMedia(w http.ResponseWriter, r *http.Request, objectID string, params v24.CreateMediaParams) {
	a.provider.operation(w, r, "media.create")
}
func (a *adapter24) PublishMedia(w http.ResponseWriter, r *http.Request, objectID string, params v24.PublishMediaParams) {
	a.provider.operation(w, r, "media.publish")
}

type adapter25 struct{ provider *Provider }

var _ v25.ServerInterface = (*adapter25)(nil)

func (a *adapter25) GetMe(w http.ResponseWriter, r *http.Request, params v25.GetMeParams) {
	a.provider.operation(w, r, "me.read")
}
func (a *adapter25) GetObject(w http.ResponseWriter, r *http.Request, objectID string, params v25.GetObjectParams) {
	a.provider.operation(w, r, "object.read")
}
func (a *adapter25) ListMedia(w http.ResponseWriter, r *http.Request, objectID string, params v25.ListMediaParams) {
	a.provider.operation(w, r, "media.list")
}
func (a *adapter25) CreateMedia(w http.ResponseWriter, r *http.Request, objectID string, params v25.CreateMediaParams) {
	a.provider.operation(w, r, "media.create")
}
func (a *adapter25) PublishMedia(w http.ResponseWriter, r *http.Request, objectID string, params v25.PublishMediaParams) {
	a.provider.operation(w, r, "media.publish")
}
