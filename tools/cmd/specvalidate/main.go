// specvalidate is development tooling; it is never linked into the simulator.
package main

import (
	"context"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: specvalidate files...")
		os.Exit(2)
	}
	for _, path := range os.Args[1:] {
		err := validate(path)
		if err != nil {
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
	for route := range doc.Paths.Map() {
		if !strings.HasPrefix(route, "/"+version+"/") {
			return fmt.Errorf("route %q does not belong to version %s", route, version)
		}
	}
	return nil
}
