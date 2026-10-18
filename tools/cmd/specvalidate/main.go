// specvalidate is development tooling; it is never linked into the simulator.
package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: specvalidate files...")
		os.Exit(2)
	}
	for _, path := range os.Args[1:] {
		if err := validate(path); err != nil {
			fmt.Fprintln(os.Stderr, path, err)
			os.Exit(1)
		}
		fmt.Println("valid", path)
	}
}

// validate checks each version independently: version differences are allowed,
// but a contract must not accidentally register a different release's routes.
func validate(path string) error {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false
	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return err
	}
	if err = doc.Validate(context.Background()); err != nil {
		return err
	}
	version := filepath.Base(filepath.Dir(path))
	if doc.Info.Version != version {
		return fmt.Errorf("info.version %q does not match directory %q", doc.Info.Version, version)
	}
	contractOnly := strings.HasSuffix(filepath.Base(path), ".openapi.yaml")
	if contractOnly && doc.Extensions["x-contract-only"] != true {
		return fmt.Errorf("full contracts require x-contract-only: true")
	}
	provider, err := contractProvider(path, doc.Extensions)
	if err != nil {
		return err
	}
	kind, _ := doc.Extensions["x-contract-kind"].(string)
	if kind != "" && kind != "graph" && kind != "oauth" && kind != "webhook" && kind != "upload" {
		return fmt.Errorf("unknown x-contract-kind %q", kind)
	}
	if !contractOnly && kind != "" {
		return fmt.Errorf("unversioned exceptions require a contract-only document")
	}
	if kind == "upload" && len(doc.Servers) == 0 {
		return fmt.Errorf("upload contracts require an explicit upload server")
	}
	if marker, present := doc.Extensions["x-dynamic-upload-url"]; present {
		if provider != "tiktok" || kind != "upload" || version != "v2" || marker != true {
			return fmt.Errorf("x-dynamic-upload-url: true is only allowed for TikTok v2 upload contracts")
		}
	}
	if provider == "tiktok" && kind == "upload" && doc.Extensions["x-dynamic-upload-url"] != true {
		return fmt.Errorf("TikTok upload contracts require x-dynamic-upload-url: true")
	}
	if contractOnly {
		if err := validateServers(doc.Servers, provider, kind); err != nil {
			return err
		}
	}
	ids := make(map[string]bool)
	for route, item := range doc.Paths.Map() {
		if !allowedRoute(provider, kind, version, route) {
			return fmt.Errorf("route %q does not belong to version %s or its explicit contract kind", route, version)
		}
		if contractOnly {
			if err := validateServers(item.Servers, provider, kind); err != nil {
				return err
			}
		}
		for method, op := range item.Operations() {
			if !contractOnly {
				continue
			}
			if op.OperationID == "" || ids[op.OperationID] {
				return fmt.Errorf("%s %s: missing or duplicate operationId %q", method, route, op.OperationID)
			}
			ids[op.OperationID] = true
			if op.Extensions["x-behavior-provenance"] != "documentation-based" {
				return fmt.Errorf("%s: require documentation-based provenance", op.OperationID)
			}
			if op.Extensions["x-simulator-implemented"] != false {
				return fmt.Errorf("%s: require x-simulator-implemented: false", op.OperationID)
			}
			sources, ok := op.Extensions["x-sources"].([]any)
			if !ok || len(sources) == 0 {
				return fmt.Errorf("%s: require nonempty x-sources array", op.OperationID)
			}
			for _, source := range sources {
				s, ok := source.(string)
				if !ok || !safeSource(provider, s) {
					return fmt.Errorf("%s: invalid source %v", op.OperationID, source)
				}
			}
			if op.Servers != nil {
				if err := validateServers(*op.Servers, provider, kind); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Existing Instagram contracts predate x-provider. New provider directories must
// name their provider explicitly, and a marker cannot change a directory's scope.
func contractProvider(path string, extensions map[string]any) (string, error) {
	provider := "instagram"
	if value, present := extensions["x-provider"]; present {
		var ok bool
		provider, ok = value.(string)
		if !ok || (provider != "instagram" && provider != "facebook" && provider != "tiktok") {
			return "", fmt.Errorf("unknown x-provider %q", value)
		}
	}
	directory := filepath.Base(filepath.Dir(filepath.Dir(path)))
	switch directory {
	case "instagram", "facebook", "tiktok":
		if provider != directory {
			return "", fmt.Errorf("x-provider %q does not match provider directory %q", provider, directory)
		}
	}
	return provider, nil
}

// Exceptions belong to one provider and contract kind. In particular, TikTok's
// returned upload URL is not a general exemption from versioned Meta routes.
func allowedRoute(provider, kind, version, route string) bool {
	if kind == "upload" {
		switch provider {
		case "instagram":
			return route == "/ig-api-upload/"+version+"/{containerID}"
		case "facebook":
			return route == "/video-upload/"+version+"/{videoID}"
		case "tiktok":
			return version == "v2" && route == "/{uploadPath}"
		}
	}
	if provider != "instagram" && kind == "webhook" {
		return route == "/webhooks/"+provider
	}
	return strings.HasPrefix(route, "/"+version+"/") || allowedUnversioned(provider, kind, route)
}

func allowedUnversioned(provider, kind, route string) bool {
	// Preserve the previously supported Instagram OAuth and receiver routes.
	// Facebook OAuth and TikTok OAuth already carry their release in the path.
	if provider != "instagram" {
		return false
	}
	switch kind {
	case "oauth":
		switch route {
		case "/oauth/authorize", "/oauth/access_token", "/access_token", "/refresh_access_token":
			return true
		}
	case "webhook":
		return route == "/webhooks/instagram"
	}
	return false
}

func validateServers(servers openapi3.Servers, provider, kind string) error {
	for _, server := range servers {
		if kind == "upload" {
			allowed := server.URL == "https://rupload.facebook.com" && (provider == "instagram" || provider == "facebook")
			if provider == "tiktok" {
				allowed = server.URL == "https://open-upload.tiktokapis.com" || server.URL == "https://upload.us.tiktokapis.com"
			}
			if !allowed {
				return fmt.Errorf("unapproved %s upload server %q", provider, server.URL)
			}
			continue
		}
		if provider == "instagram" && kind == "webhook" && server.URL == "http://localhost:3000" {
			continue
		}
		u, err := url.Parse(server.URL)
		if err != nil || u.Scheme != "https" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Port() != "" {
			return fmt.Errorf("unsafe server URL %q", server.URL)
		}
		if provider != "instagram" && u.Path != "" {
			return fmt.Errorf("%s server URL must not contain a path: %q", provider, server.URL)
		}
		if !allowedServerHost(provider, kind, u.Host) {
			return fmt.Errorf("unapproved %s %s server hostname %q", provider, kind, u.Host)
		}
	}
	return nil
}

func allowedServerHost(provider, kind, host string) bool {
	switch provider {
	case "instagram":
		// Keep the existing Instagram allowlist unchanged.
		switch host {
		case "graph.facebook.com", "graph.instagram.com", "api.instagram.com", "www.instagram.com", "www.facebook.com", "example.invalid":
			return true
		}
	case "facebook":
		switch kind {
		case "", "graph":
			return host == "graph.facebook.com" || host == "graph-video.facebook.com"
		case "oauth":
			return host == "graph.facebook.com" || host == "www.facebook.com"
		case "webhook":
			return host == "example.invalid"
		}
	case "tiktok":
		switch kind {
		case "", "graph":
			return host == "open.tiktokapis.com"
		case "oauth":
			return host == "open.tiktokapis.com" || host == "www.tiktok.com"
		case "webhook":
			return host == "example.invalid"
		}
	}
	return false
}

func safeSource(provider, source string) bool {
	u, err := url.Parse(source)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return false
	}
	if provider == "tiktok" {
		return u.Host == "developers.tiktok.com" || (u.Host == "github.com" && strings.HasPrefix(u.Path, "/tiktok/"))
	}
	if provider != "instagram" && provider != "facebook" {
		return false
	}
	switch u.Host {
	case "developers.facebook.com", "developers.meta.com":
		return true
	case "www.postman.com":
		return strings.HasPrefix(u.Path, "/meta/")
	case "github.com", "raw.githubusercontent.com":
		return strings.HasPrefix(u.Path, "/fbsamples/") || strings.HasPrefix(u.Path, "/facebook/") || strings.HasPrefix(u.Path, "/facebookarchive/") || strings.HasPrefix(u.Path, "/instagram/")
	}
	return false
}
