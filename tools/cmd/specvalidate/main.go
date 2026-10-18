// specvalidate is development tooling; it is never linked into the simulator.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: specvalidate files...")
		os.Exit(2)
	}
	var reference []byte
	for _, path := range os.Args[1:] {
		loader := openapi3.NewLoader()
		loader.IsExternalRefsAllowed = false
		doc, err := loader.LoadFromFile(path)
		if err == nil {
			err = doc.Validate(context.Background())
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, path, err)
			os.Exit(1)
		}
		// This command validates the shared Instagram version family. Reject drift
		// until a version gets a separate generated interface/implementation.
		doc.Info.Version = ""
		normalized, err := json.Marshal(doc)
		if err != nil {
			panic(err)
		}
		if reference != nil && !bytes.Equal(reference, normalized) {
			fmt.Fprintln(os.Stderr, path, "differs from the shared version contract; generate a separate implementation before diverging")
			os.Exit(1)
		}
		reference = normalized
		fmt.Println("valid", path)
	}
}
