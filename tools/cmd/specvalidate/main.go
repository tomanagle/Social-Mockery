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
	kind, _ := doc.Extensions["x-contract-kind"].(string)
	if kind != "" && kind != "graph" && kind != "oauth" && kind != "webhook" {
		return fmt.Errorf("unknown x-contract-kind %q", kind)
	}
	if !contractOnly && kind != "" {
		return fmt.Errorf("unversioned exceptions require a contract-only document")
	}
	if contractOnly {
		if err := validateServers(doc.Servers, kind); err != nil {
			return err
		}
	}
	ids := make(map[string]bool)
	for route, item := range doc.Paths.Map() {
		if !strings.HasPrefix(route, "/"+version+"/") && !allowedUnversioned(kind, route) {
			return fmt.Errorf("route %q does not belong to version %s or its explicit contract kind", route, version)
		}
		if contractOnly {
			if err := validateServers(item.Servers, kind); err != nil {
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
				if !ok || !safeSource(s) {
					return fmt.Errorf("%s: invalid source %v", op.OperationID, source)
				}
			}
			if op.Servers != nil {
				if err := validateServers(*op.Servers, kind); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// These exceptions are provider OAuth endpoints and our example receiver path,
// never a general exemption from versioned route validation.
func allowedUnversioned(kind, route string) bool {
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

func validateServers(servers openapi3.Servers, kind string) error {
	for _, server := range servers {
		if kind == "webhook" && server.URL == "http://localhost:3000" {
			continue
		}
		u, err := url.Parse(server.URL)
		if err != nil || u.Scheme != "https" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Port() != "" {
			return fmt.Errorf("unsafe server URL %q", server.URL)
		}
		switch u.Host {
		case "graph.facebook.com", "graph.instagram.com", "api.instagram.com", "www.instagram.com", "www.facebook.com", "example.invalid":
		default:
			return fmt.Errorf("unapproved server hostname %q", u.Host)
		}
	}
	return nil
}

func safeSource(source string) bool {
	u, err := url.Parse(source)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Port() != "" {
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
