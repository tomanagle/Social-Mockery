// Package meta implements a deliberately small, documented Graph API subset.
package meta

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tomanagle/Social-Mockery/internal/simulation"
	"github.com/tomanagle/Social-Mockery/internal/store"
	"github.com/tomanagle/Social-Mockery/internal/telemetry"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const MaxBody = 1 << 20

type Provider struct {
	engine    *simulation.Engine
	routers   map[string]http.Handler
	telemetry *telemetry.Telemetry
}

func New(e *simulation.Engine, instrumentation ...*telemetry.Telemetry) *Provider {
	t := telemetry.Noop()
	if len(instrumentation) > 0 && instrumentation[0] != nil {
		t = instrumentation[0]
	}
	p := &Provider{engine: e, telemetry: t}
	p.routers = versionRouters(p)
	return p
}

type responseStatus struct {
	http.ResponseWriter
	status int
}

func (w *responseStatus) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
func (p *Provider) operation(w http.ResponseWriter, r *http.Request, name string) {
	started := time.Now()
	ctx, span := p.telemetry.Start(r.Context(), "instagram."+name)
	recorded := &responseStatus{ResponseWriter: w, status: 200}
	defer func() {
		outcome := "success"
		var err error
		expected := false
		if recorded.status >= 400 {
			outcome = "error"
			err = errors.New("provider operation failed")
			expected = recorded.status < 500 || recorded.status == 507
			if recorded.status == 507 {
				telemetry.ExpectedFailure(ctx)
			}
			if expected {
				p.telemetry.SimulatedError(ctx, "instagram", "request_rejected")
			}
		}
		telemetry.Finish(span, err, expected)
		p.telemetry.ObserveOperation(ctx, name, "instagram", outcome, time.Since(started))
		if name == "media.publish" {
			p.telemetry.Publish(ctx, "instagram", outcome)
		}
	}()
	p.serveGraph(recorded, r.WithContext(ctx))
}
func (p *Provider) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	u, err := url.Parse(strings.TrimPrefix(r.RequestURI, "/"))
	if err != nil || u.Scheme != "https" || u.User != nil || u.Fragment != "" || (u.Host != "graph.instagram.com" && u.Host != "graph.facebook.com") {
		failure(w, 400, 100, "Unsupported original URL; only HTTPS graph.instagram.com and graph.facebook.com are allowed")
		return
	}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if strings.Contains(u.EscapedPath(), "%") || len(parts) < 2 || len(parts) > 3 {
		failure(w, 404, 100, "Unsupported endpoint")
		return
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			failure(w, 404, 100, "Unsupported endpoint")
			return
		}
	}
	router, supportedVersion := p.routers[parts[0]]
	if !supportedVersion {
		failure(w, 404, 100, "Unsupported API version")
		return
	}
	if r.Method != "GET" && r.Method != "POST" {
		failure(w, 405, 100, "Unsupported method")
		return
	}
	supported := (len(parts) == 2 && r.Method == "GET") || (len(parts) == 3 && parts[2] == "media") || (len(parts) == 3 && parts[2] == "media_publish" && r.Method == "POST")
	if !supported {
		failure(w, 404, 100, "Unsupported endpoint; see docs/compatibility.md")
		return
	}
	clone := r.Clone(r.Context())
	clone.URL = &url.URL{Path: u.Path, RawQuery: u.RawQuery}
	router.ServeHTTP(w, clone)
}
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func failure(w http.ResponseWriter, status, code int, message string) {
	JSON(w, status, errorResponse{Error: graphError{Message: message, Type: "IGApiException", Code: code}})
}
func failStore(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		failure(w, 400, 100, "Unsupported request: object does not exist or is not accessible")
	case errors.Is(err, store.ErrConflict):
		failure(w, 400, 9007, "Container is not ready, expired, or already published")
	case errors.Is(err, store.ErrCapacity):
		failure(w, 507, 1, err.Error())
	default:
		failure(w, 500, 1, "Simulator storage failure")
	}
}

// ServeHTTP parses the original URL directly. It never creates an outbound request.
// This handler must run before ServeMux's path cleaning (which changes https://).
func (p *Provider) serveGraph(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimPrefix(r.RequestURI, "/")
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Fragment != "" || (u.Host != "graph.instagram.com" && u.Host != "graph.facebook.com") {
		failure(w, 400, 100, "Unsupported original URL; only HTTPS graph.instagram.com and graph.facebook.com are allowed")
		return
	}
	if strings.Contains(u.EscapedPath(), "%") {
		failure(w, 400, 100, "Encoded endpoint paths are unsupported")
		return
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || len(parts) > 3 || p.routers[parts[0]] == nil {
		failure(w, 404, 100, "Unsupported API version or endpoint; see docs/compatibility.md")
		return
	}
	if r.Method != "GET" && r.Method != "POST" {
		failure(w, 405, 100, "Unsupported method")
		return
	}
	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		failure(w, 400, 100, "Malformed query parameters")
		return
	}
	values, err := parameters(w, r, query)
	if err != nil {
		failure(w, 400, 100, err.Error())
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	edgeName := ""
	if len(parts) == 3 {
		edgeName = parts[2]
	}
	allowed := map[string]bool{"access_token": true}
	if r.Method == "GET" {
		allowed["fields"] = true
		if edgeName == "media" {
			allowed["limit"] = true
			allowed["after"] = true
		}
	}
	if r.Method == "POST" && edgeName == "media" {
		for _, key := range []string{"image_url", "video_url", "media_type", "caption"} {
			allowed[key] = true
		}
	}
	if r.Method == "POST" && edgeName == "media_publish" {
		allowed["creation_id"] = true
	}
	for key := range values {
		if !allowed[key] {
			failure(w, 400, 100, "Unsupported parameter: "+key)
			return
		}
	}

	p.engine.Mu.Lock()
	defer p.engine.Mu.Unlock()
	t, now, ok := p.authorize(w, r, values, u.Host)
	if !ok {
		return
	}

	id := parts[1]
	edge := ""
	if len(parts) == 3 {
		edge = parts[2]
	}
	if edge != "" && id != t.AccountID {
		failure(w, 400, 100, "Account is not accessible with this token")
		return
	}
	if r.Method == "POST" && edge == "media" {
		p.create(w, r, id, values)
		return
	}
	if r.Method == "POST" && edge == "media_publish" {
		if values.Get("creation_id") == "" {
			failure(w, 400, 100, "creation_id is required")
			return
		}
		post, err := p.engine.Store.Publish(r.Context(), id, values.Get("creation_id"), p.engine.MaxRecords)
		if err != nil {
			failStore(w, err)
			return
		}
		JSON(w, 200, idResponse{ID: post.ID})
		return
	}
	if r.Method == "GET" && edge == "media" {
		p.list(w, r, id, values, u)
		return
	}
	if r.Method == "GET" && edge == "" {
		p.get(w, r, id, t.AccountID, values, now)
		return
	}
	failure(w, 404, 100, "Unsupported endpoint; see docs/compatibility.md")
}
func parameters(w http.ResponseWriter, r *http.Request, query url.Values) (url.Values, error) {
	r.Body = http.MaxBytesReader(w, r.Body, MaxBody)
	body := url.Values{}
	if r.Method == "POST" {
		typ, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil && r.ContentLength != 0 {
			return nil, fmt.Errorf("invalid Content-Type")
		}
		switch typ {
		case "application/json":
			d := json.NewDecoder(r.Body)
			opening, err := d.Token()
			if err != nil || opening != json.Delim('{') {
				return nil, fmt.Errorf("JSON object required")
			}
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return nil, fmt.Errorf("invalid JSON body")
				}
				k, ok := key.(string)
				if !ok || body.Has(k) {
					return nil, fmt.Errorf("duplicate or invalid JSON key")
				}
				var value *string
				if err = d.Decode(&value); err != nil || value == nil {
					return nil, fmt.Errorf("parameter %s must be a string", k)
				}
				body.Set(k, *value)
			}
			if closing, err := d.Token(); err != nil || closing != json.Delim('}') {
				return nil, fmt.Errorf("invalid JSON body")
			}
			var extra any
			if d.Decode(&extra) != io.EOF {
				return nil, fmt.Errorf("one JSON object required")
			}

		case "application/x-www-form-urlencoded":
			if err = r.ParseForm(); err != nil {
				return nil, fmt.Errorf("invalid form body: %w", err)
			}
			body = r.PostForm
		case "multipart/form-data":
			if err = r.ParseMultipartForm(MaxBody); err != nil {
				if r.MultipartForm != nil {
					r.MultipartForm.RemoveAll()
				}
				return nil, fmt.Errorf("invalid multipart body")
			}
			if len(r.MultipartForm.File) > 0 {
				r.MultipartForm.RemoveAll()
				return nil, fmt.Errorf("binary uploads unsupported; use image_url or video_url")
			}
			body = r.MultipartForm.Value
		case "":
			if r.ContentLength != 0 {
				return nil, fmt.Errorf("Content-Type required")
			}
		default:
			return nil, fmt.Errorf("unsupported Content-Type")
		}
	}
	for k, v := range query {
		if len(v) != 1 {
			return nil, fmt.Errorf("duplicate parameter %s", k)
		}
	}
	for k, v := range body {
		if len(v) != 1 || len(query[k]) > 0 {
			return nil, fmt.Errorf("duplicate parameter %s", k)
		}
		query[k] = v
	}
	return query, nil
}
func has(scopes []string, s string) bool {
	for _, x := range scopes {
		if x == s {
			return true
		}
	}
	return false
}
func (p *Provider) create(w http.ResponseWriter, r *http.Request, id string, v url.Values) {
	a, err := p.engine.Store.Account(r.Context(), id)
	if err != nil {
		failStore(w, err)
		return
	}
	if a.Type == "PERSONAL" {
		failure(w, 403, 200, "Publishing requires a professional Instagram account")
		return
	}
	kind := v.Get("media_type")
	mediaURL := v.Get("image_url")
	switch kind {
	case "", "IMAGE":
		kind = "IMAGE"
		if v.Get("video_url") != "" {
			failure(w, 400, 100, "video_url requires media_type=REELS")
			return
		}
	case "REELS":
		mediaURL = v.Get("video_url")
		if v.Get("image_url") != "" {
			failure(w, 400, 100, "image_url is not supported for REELS")
			return
		}
	default:
		failure(w, 400, 100, "Unsupported media_type; Phase 1 supports IMAGE and REELS")
		return
	}
	parsed, err := url.Parse(mediaURL)
	if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || len(mediaURL) > 8192 {
		failure(w, 400, 100, "A valid image_url or video_url is required")
		return
	}
	if len([]rune(v.Get("caption"))) > 2200 {
		failure(w, 400, 100, "caption exceeds 2200 characters")
		return
	}
	c, err := p.engine.Store.CreateContainer(r.Context(), store.Container{AccountID: id, MediaType: kind, URL: mediaURL, Caption: v.Get("caption")}, p.engine.MaxRecords)
	if err != nil {
		failStore(w, err)
		return
	}
	p.telemetry.Processing(r.Context(), float64(c.ReadyAt-c.CreatedAt))
	JSON(w, 200, idResponse{ID: c.ID})
}
func projection(w http.ResponseWriter, fields string, all map[string]any) {
	if fields == "" {
		JSON(w, 200, all)
		return
	}
	out := map[string]any{"id": all["id"]}
	for _, field := range strings.Split(fields, ",") {
		field = strings.TrimSpace(field)
		v, ok := all[field]
		if !ok {
			failure(w, 400, 100, "Unsupported field: "+field)
			return
		}
		out[field] = v
	}
	JSON(w, 200, out)
}
func postFields(post store.Post) map[string]any {
	kind, product := post.MediaType, "FEED"
	if kind == "REELS" {
		kind = "VIDEO"
		product = "REELS"
	}
	return map[string]any{"id": post.ID, "caption": post.Caption, "media_type": kind, "media_product_type": product, "media_url": post.URL, "thumbnail_url": post.URL, "timestamp": time.Unix(post.CreatedAt, 0).UTC().Format("2006-01-02T15:04:05+0000"), "permalink": "https://www.instagram.com/p/mock_" + post.ID + "/"}
}
func (p *Provider) get(w http.ResponseWriter, r *http.Request, id, owner string, v url.Values, now int64) {
	if id == owner {
		a, err := p.engine.Store.Account(r.Context(), id)
		if err != nil {
			failStore(w, err)
			return
		}
		projection(w, v.Get("fields"), map[string]any{"id": a.ID, "username": a.Username, "account_type": a.Type})
		return
	}
	c, err := p.engine.Store.Container(r.Context(), id)
	if err == nil {
		if c.AccountID != owner {
			failure(w, 400, 100, "Object is not accessible")
			return
		}
		status := simulation.Status(c, now)
		projection(w, v.Get("fields"), map[string]any{"id": c.ID, "status_code": status, "status": status})
		return
	}
	if !errors.Is(err, store.ErrNotFound) {
		failStore(w, err)
		return
	}
	post, err := p.engine.Store.Post(r.Context(), id)
	if err != nil {
		failStore(w, err)
		return
	}
	if post.AccountID != owner {
		failure(w, 400, 100, "Object is not accessible")
		return
	}
	projection(w, v.Get("fields"), postFields(post))
}
func (p *Provider) list(w http.ResponseWriter, r *http.Request, id string, v url.Values, u *url.URL) {
	if f := v.Get("fields"); f != "" {
		supported := postFields(store.Post{})
		for _, field := range strings.Split(f, ",") {
			if _, ok := supported[strings.TrimSpace(field)]; !ok {
				failure(w, 400, 100, "Unsupported field: "+field)
				return
			}
		}
	}

	limit := 25
	offset := 0
	var err error
	if v.Get("limit") != "" {
		limit, err = strconv.Atoi(v.Get("limit"))
		if err != nil || limit < 1 || limit > 100 {
			failure(w, 400, 100, "limit must be 1..100")
			return
		}
	}
	if v.Get("after") != "" {
		offset, err = strconv.Atoi(v.Get("after"))
		if err != nil || offset < 0 || offset > p.engine.MaxRecords {
			failure(w, 400, 100, "invalid after cursor")
			return
		}
	}
	posts, err := p.engine.Store.Posts(r.Context(), id, limit+1, offset)
	if err != nil {
		failStore(w, err)
		return
	}
	more := len(posts) > limit
	if more {
		posts = posts[:limit]
	}
	data := []map[string]any{}
	for _, post := range posts {
		all := postFields(post)
		if f := v.Get("fields"); f != "" {
			selected := map[string]any{"id": post.ID}
			for _, key := range strings.Split(f, ",") {
				key = strings.TrimSpace(key)
				value, ok := all[key]
				if !ok {
					failure(w, 400, 100, "Unsupported field: "+key)
					return
				}
				selected[key] = value
			}
			all = selected
		}
		data = append(data, all)
	}
	out := map[string]any{"data": data}
	if more {
		q := u.Query()
		q.Set("after", strconv.Itoa(offset+limit))
		u.RawQuery = q.Encode()
		out["paging"] = map[string]any{"cursors": map[string]string{"after": strconv.Itoa(offset + limit)}, "next": "/" + u.String()}
	}
	JSON(w, 200, out)
}

func (p *Provider) authorize(w http.ResponseWriter, r *http.Request, values url.Values, host string) (store.Token, int64, bool) {
	ctx, span := p.telemetry.Start(r.Context(), "auth.validate")
	recorded := &responseStatus{ResponseWriter: w, status: 200}
	w = recorded
	r = r.WithContext(ctx)
	defer func() {
		var err error
		if recorded.status >= 400 {
			err = errors.New("authentication rejected")
		}
		telemetry.Finish(span, err, recorded.status < 500)
	}()
	token := values.Get("access_token")
	if auth := r.Header.Get("Authorization"); auth != "" {
		if !strings.HasPrefix(auth, "Bearer ") || len(auth) <= 7 {
			failure(w, 400, 190, "Invalid authorization header")
			return store.Token{}, 0, false
		}
		header := strings.TrimPrefix(auth, "Bearer ")
		if token != "" && token != header {
			failure(w, 400, 190, "Conflicting access tokens")
			return store.Token{}, 0, false
		}
		token = header
	}
	t, err := p.engine.Store.Token(r.Context(), token)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			failure(w, 400, 190, "Invalid OAuth access token")
		} else {
			failStore(w, err)
		}
		return store.Token{}, 0, false
	}
	now, _, err := p.engine.Store.Clock(r.Context())
	if err != nil {
		failStore(w, err)
		return store.Token{}, 0, false
	}
	if t.ExpiresAt > 0 && now >= t.ExpiresAt {
		failure(w, 400, 190, "Access token has expired")
		return store.Token{}, 0, false
	}
	basic, publish := "instagram_business_basic", "instagram_business_content_publish"
	if host == "graph.facebook.com" {
		basic, publish = "instagram_basic", "instagram_content_publish"
	}
	if !has(t.Scopes, basic) || (r.Method == "POST" && !has(t.Scopes, publish)) {
		failure(w, 403, 200, "Required permission was not granted")
		return store.Token{}, 0, false
	}
	return t, now, true
}

// These internal response values are independent of generated version models.
// Version adapters may translate them when an upstream response shape changes.
type idResponse struct {
	ID string `json:"id"`
}
type graphError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    int    `json:"code"`
}
type errorResponse struct {
	Error graphError `json:"error"`
}
