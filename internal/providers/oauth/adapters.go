// Package adapters bind the implemented OAuth slice to generated contracts.
// Unimplemented reference operations are deliberately not exposed by Match.
package oauth

import (
	facebookv24 "github.com/tomanagle/Social-Mockery/internal/contracts/facebook/v24/oauth"
	facebookv25 "github.com/tomanagle/Social-Mockery/internal/contracts/facebook/v25/oauth"
	facebookv26 "github.com/tomanagle/Social-Mockery/internal/contracts/facebook/v26/oauth"
	instagramv24 "github.com/tomanagle/Social-Mockery/internal/contracts/instagram/v24/oauth"
	instagramv25 "github.com/tomanagle/Social-Mockery/internal/contracts/instagram/v25/oauth"
	instagramv26 "github.com/tomanagle/Social-Mockery/internal/contracts/instagram/v26/oauth"
	tiktokv2 "github.com/tomanagle/Social-Mockery/internal/contracts/tiktok/v2/oauth"
	"net/http"
	"net/url"
	"strings"
)

type instagramv24Adapter struct{ p *Provider }

var _ instagramv24.ServerInterface = (*instagramv24Adapter)(nil)

func (a *instagramv24Adapter) ExchangeInstagramLongLivedToken(w http.ResponseWriter, r *http.Request, params instagramv24.ExchangeInstagramLongLivedTokenParams) {
	a.p.instagramLongLived(w, r, false)
}
func (a *instagramv24Adapter) ExchangeInstagramAuthorizationCode(w http.ResponseWriter, r *http.Request) {
	a.p.serve(w, r)
}
func (a *instagramv24Adapter) AuthorizeInstagramBusinessLogin(w http.ResponseWriter, r *http.Request, params instagramv24.AuthorizeInstagramBusinessLoginParams) {
	a.p.serve(w, r)
}
func (a *instagramv24Adapter) RefreshInstagramLongLivedToken(w http.ResponseWriter, r *http.Request, params instagramv24.RefreshInstagramLongLivedTokenParams) {
	a.p.instagramLongLived(w, r, true)
}
func (a *instagramv24Adapter) DebugFacebookAccessToken(w http.ResponseWriter, r *http.Request, params instagramv24.DebugFacebookAccessTokenParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *instagramv24Adapter) AuthorizeFacebookLogin(w http.ResponseWriter, r *http.Request, params instagramv24.AuthorizeFacebookLoginParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *instagramv24Adapter) RevokeFacebookAuthorizationSelf(w http.ResponseWriter, r *http.Request, params instagramv24.RevokeFacebookAuthorizationSelfParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *instagramv24Adapter) GetFacebookPermissionsSelf(w http.ResponseWriter, r *http.Request, params instagramv24.GetFacebookPermissionsSelfParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *instagramv24Adapter) ExchangeFacebookAccessToken(w http.ResponseWriter, r *http.Request, params instagramv24.ExchangeFacebookAccessTokenParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *instagramv24Adapter) CreateFacebookClientCode(w http.ResponseWriter, r *http.Request, params instagramv24.CreateFacebookClientCodeParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *instagramv24Adapter) RevokeFacebookAuthorizationById(w http.ResponseWriter, r *http.Request, objectID string, params instagramv24.RevokeFacebookAuthorizationByIdParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *instagramv24Adapter) GetFacebookPermissionsById(w http.ResponseWriter, r *http.Request, objectID string, params instagramv24.GetFacebookPermissionsByIdParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *instagramv24Adapter) RevokeFacebookPermission(w http.ResponseWriter, r *http.Request, objectID string, permissionName string, params instagramv24.RevokeFacebookPermissionParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}

type instagramv25Adapter struct{ p *Provider }

var _ instagramv25.ServerInterface = (*instagramv25Adapter)(nil)

func (a *instagramv25Adapter) ExchangeInstagramLongLivedToken(w http.ResponseWriter, r *http.Request, params instagramv25.ExchangeInstagramLongLivedTokenParams) {
	a.p.instagramLongLived(w, r, false)
}
func (a *instagramv25Adapter) ExchangeInstagramAuthorizationCode(w http.ResponseWriter, r *http.Request) {
	a.p.serve(w, r)
}
func (a *instagramv25Adapter) AuthorizeInstagramBusinessLogin(w http.ResponseWriter, r *http.Request, params instagramv25.AuthorizeInstagramBusinessLoginParams) {
	a.p.serve(w, r)
}
func (a *instagramv25Adapter) RefreshInstagramLongLivedToken(w http.ResponseWriter, r *http.Request, params instagramv25.RefreshInstagramLongLivedTokenParams) {
	a.p.instagramLongLived(w, r, true)
}
func (a *instagramv25Adapter) DebugFacebookAccessToken(w http.ResponseWriter, r *http.Request, params instagramv25.DebugFacebookAccessTokenParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *instagramv25Adapter) AuthorizeFacebookLogin(w http.ResponseWriter, r *http.Request, params instagramv25.AuthorizeFacebookLoginParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *instagramv25Adapter) RevokeFacebookAuthorizationSelf(w http.ResponseWriter, r *http.Request, params instagramv25.RevokeFacebookAuthorizationSelfParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *instagramv25Adapter) GetFacebookPermissionsSelf(w http.ResponseWriter, r *http.Request, params instagramv25.GetFacebookPermissionsSelfParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *instagramv25Adapter) ExchangeFacebookAccessToken(w http.ResponseWriter, r *http.Request, params instagramv25.ExchangeFacebookAccessTokenParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *instagramv25Adapter) CreateFacebookClientCode(w http.ResponseWriter, r *http.Request, params instagramv25.CreateFacebookClientCodeParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *instagramv25Adapter) RevokeFacebookAuthorizationById(w http.ResponseWriter, r *http.Request, objectID string, params instagramv25.RevokeFacebookAuthorizationByIdParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *instagramv25Adapter) GetFacebookPermissionsById(w http.ResponseWriter, r *http.Request, objectID string, params instagramv25.GetFacebookPermissionsByIdParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *instagramv25Adapter) RevokeFacebookPermission(w http.ResponseWriter, r *http.Request, objectID string, permissionName string, params instagramv25.RevokeFacebookPermissionParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}

type instagramv26Adapter struct{ p *Provider }

var _ instagramv26.ServerInterface = (*instagramv26Adapter)(nil)

func (a *instagramv26Adapter) ExchangeInstagramLongLivedToken(w http.ResponseWriter, r *http.Request, params instagramv26.ExchangeInstagramLongLivedTokenParams) {
	a.p.instagramLongLived(w, r, false)
}
func (a *instagramv26Adapter) ExchangeInstagramAuthorizationCode(w http.ResponseWriter, r *http.Request) {
	a.p.serve(w, r)
}
func (a *instagramv26Adapter) AuthorizeInstagramBusinessLogin(w http.ResponseWriter, r *http.Request, params instagramv26.AuthorizeInstagramBusinessLoginParams) {
	a.p.serve(w, r)
}
func (a *instagramv26Adapter) RefreshInstagramLongLivedToken(w http.ResponseWriter, r *http.Request, params instagramv26.RefreshInstagramLongLivedTokenParams) {
	a.p.instagramLongLived(w, r, true)
}
func (a *instagramv26Adapter) DebugFacebookAccessToken(w http.ResponseWriter, r *http.Request, params instagramv26.DebugFacebookAccessTokenParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *instagramv26Adapter) AuthorizeFacebookLogin(w http.ResponseWriter, r *http.Request, params instagramv26.AuthorizeFacebookLoginParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *instagramv26Adapter) RevokeFacebookAuthorizationSelf(w http.ResponseWriter, r *http.Request, params instagramv26.RevokeFacebookAuthorizationSelfParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *instagramv26Adapter) GetFacebookPermissionsSelf(w http.ResponseWriter, r *http.Request, params instagramv26.GetFacebookPermissionsSelfParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *instagramv26Adapter) ExchangeFacebookAccessToken(w http.ResponseWriter, r *http.Request, params instagramv26.ExchangeFacebookAccessTokenParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *instagramv26Adapter) CreateFacebookClientCode(w http.ResponseWriter, r *http.Request, params instagramv26.CreateFacebookClientCodeParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *instagramv26Adapter) RevokeFacebookAuthorizationById(w http.ResponseWriter, r *http.Request, objectID string, params instagramv26.RevokeFacebookAuthorizationByIdParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *instagramv26Adapter) GetFacebookPermissionsById(w http.ResponseWriter, r *http.Request, objectID string, params instagramv26.GetFacebookPermissionsByIdParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *instagramv26Adapter) RevokeFacebookPermission(w http.ResponseWriter, r *http.Request, objectID string, permissionName string, params instagramv26.RevokeFacebookPermissionParams) {
	fail(w, "instagram", 501, "unsupported_operation", "This reference operation is not implemented")
}

type facebookv24Adapter struct{ p *Provider }

var _ facebookv24.ServerInterface = (*facebookv24Adapter)(nil)

func (a *facebookv24Adapter) DebugFacebookToken(w http.ResponseWriter, r *http.Request, params facebookv24.DebugFacebookTokenParams) {
	fail(w, "facebook", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *facebookv24Adapter) AuthorizeFacebook(w http.ResponseWriter, r *http.Request, params facebookv24.AuthorizeFacebookParams) {
	a.p.serve(w, r)
}
func (a *facebookv24Adapter) ExchangeFacebookToken(w http.ResponseWriter, r *http.Request, params facebookv24.ExchangeFacebookTokenParams) {
	a.p.serve(w, r)
}
func (a *facebookv24Adapter) RevokeFacebookPermissions(w http.ResponseWriter, r *http.Request, userID string) {
	fail(w, "facebook", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *facebookv24Adapter) ReadFacebookPermissions(w http.ResponseWriter, r *http.Request, userID string) {
	fail(w, "facebook", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *facebookv24Adapter) RevokeFacebookPermission(w http.ResponseWriter, r *http.Request, userID string, permission string) {
	fail(w, "facebook", 501, "unsupported_operation", "This reference operation is not implemented")
}

type facebookv25Adapter struct{ p *Provider }

var _ facebookv25.ServerInterface = (*facebookv25Adapter)(nil)

func (a *facebookv25Adapter) DebugFacebookToken(w http.ResponseWriter, r *http.Request, params facebookv25.DebugFacebookTokenParams) {
	fail(w, "facebook", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *facebookv25Adapter) AuthorizeFacebook(w http.ResponseWriter, r *http.Request, params facebookv25.AuthorizeFacebookParams) {
	a.p.serve(w, r)
}
func (a *facebookv25Adapter) ExchangeFacebookToken(w http.ResponseWriter, r *http.Request, params facebookv25.ExchangeFacebookTokenParams) {
	a.p.serve(w, r)
}
func (a *facebookv25Adapter) RevokeFacebookPermissions(w http.ResponseWriter, r *http.Request, userID string) {
	fail(w, "facebook", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *facebookv25Adapter) ReadFacebookPermissions(w http.ResponseWriter, r *http.Request, userID string) {
	fail(w, "facebook", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *facebookv25Adapter) RevokeFacebookPermission(w http.ResponseWriter, r *http.Request, userID string, permission string) {
	fail(w, "facebook", 501, "unsupported_operation", "This reference operation is not implemented")
}

type facebookv26Adapter struct{ p *Provider }

var _ facebookv26.ServerInterface = (*facebookv26Adapter)(nil)

func (a *facebookv26Adapter) DebugFacebookToken(w http.ResponseWriter, r *http.Request, params facebookv26.DebugFacebookTokenParams) {
	fail(w, "facebook", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *facebookv26Adapter) AuthorizeFacebook(w http.ResponseWriter, r *http.Request, params facebookv26.AuthorizeFacebookParams) {
	a.p.serve(w, r)
}
func (a *facebookv26Adapter) ExchangeFacebookToken(w http.ResponseWriter, r *http.Request, params facebookv26.ExchangeFacebookTokenParams) {
	a.p.serve(w, r)
}
func (a *facebookv26Adapter) RevokeFacebookPermissions(w http.ResponseWriter, r *http.Request, userID string) {
	fail(w, "facebook", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *facebookv26Adapter) ReadFacebookPermissions(w http.ResponseWriter, r *http.Request, userID string) {
	fail(w, "facebook", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *facebookv26Adapter) RevokeFacebookPermission(w http.ResponseWriter, r *http.Request, userID string, permission string) {
	fail(w, "facebook", 501, "unsupported_operation", "This reference operation is not implemented")
}

type tiktokv2Adapter struct{ p *Provider }

var _ tiktokv2.ServerInterface = (*tiktokv2Adapter)(nil)

func (a *tiktokv2Adapter) AuthorizeTikTok(w http.ResponseWriter, r *http.Request, params tiktokv2.AuthorizeTikTokParams) {
	a.p.serve(w, r)
}
func (a *tiktokv2Adapter) CheckTikTokQRCode(w http.ResponseWriter, r *http.Request) {
	fail(w, "tiktok", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *tiktokv2Adapter) CreateTikTokQRCode(w http.ResponseWriter, r *http.Request) {
	fail(w, "tiktok", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *tiktokv2Adapter) RevokeTikTokAccess(w http.ResponseWriter, r *http.Request) {
	fail(w, "tiktok", 501, "unsupported_operation", "This reference operation is not implemented")
}
func (a *tiktokv2Adapter) ExchangeOrRefreshUserToken(w http.ResponseWriter, r *http.Request) {
	a.p.serve(w, r)
}
func contractRouters(p *Provider) map[string]http.Handler {
	return map[string]http.Handler{
		"instagram/v24": instagramv24.HandlerWithOptions(&instagramv24Adapter{p: p}, instagramv24.StdHTTPServerOptions{ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			fail(w, "instagram", 400, "invalid_request", "Missing or malformed OAuth parameters")
		}}),
		"instagram/v25": instagramv25.HandlerWithOptions(&instagramv25Adapter{p: p}, instagramv25.StdHTTPServerOptions{ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			fail(w, "instagram", 400, "invalid_request", "Missing or malformed OAuth parameters")
		}}),
		"instagram/v26": instagramv26.HandlerWithOptions(&instagramv26Adapter{p: p}, instagramv26.StdHTTPServerOptions{ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			fail(w, "instagram", 400, "invalid_request", "Missing or malformed OAuth parameters")
		}}),
		"facebook/v24": facebookv24.HandlerWithOptions(&facebookv24Adapter{p: p}, facebookv24.StdHTTPServerOptions{ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			fail(w, "facebook", 400, "invalid_request", "Missing or malformed OAuth parameters")
		}}),
		"facebook/v25": facebookv25.HandlerWithOptions(&facebookv25Adapter{p: p}, facebookv25.StdHTTPServerOptions{ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			fail(w, "facebook", 400, "invalid_request", "Missing or malformed OAuth parameters")
		}}),
		"facebook/v26": facebookv26.HandlerWithOptions(&facebookv26Adapter{p: p}, facebookv26.StdHTTPServerOptions{ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			fail(w, "facebook", 400, "invalid_request", "Missing or malformed OAuth parameters")
		}}),
		"tiktok/v2": tiktokv2.HandlerWithOptions(&tiktokv2Adapter{p: p}, tiktokv2.StdHTTPServerOptions{ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			fail(w, "tiktok", 400, "invalid_request", "Missing or malformed OAuth parameters")
		}}),
	}
}
func (p *Provider) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	securityHeaders(w)
	product, operation, ok := Match(r)
	if !ok {
		http.NotFound(w, r)
		return
	}
	method := "GET"
	if operation == "token" && product != "facebook" {
		method = "POST"
	}
	if r.Method != method {
		w.Header().Set("Allow", method)
		fail(w, product, 405, "invalid_request", "Unsupported method")
		return
	}
	u, _ := url.Parse(strings.TrimPrefix(r.RequestURI, "/"))
	version := "v25"
	if product == "facebook" {
		version = strings.Split(strings.TrimPrefix(u.Path, "/"), ".")[0]
	}
	if product == "tiktok" {
		version = "v2"
	}
	clone := r.Clone(r.Context())
	clone.URL = &url.URL{Path: u.Path, RawQuery: u.RawQuery}
	if product == "instagram" && u.Path == "/oauth/authorize/" {
		clone.URL.Path = "/oauth/authorize"
	}
	p.routers[product+"/"+version].ServeHTTP(w, clone)
}
