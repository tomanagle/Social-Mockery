package profiles

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMustDefault(t *testing.T) {
	if !reflect.DeepEqual(Must(""), Default()) {
		t.Fatal("Must did not return the default profile")
	}
}

func TestMustProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "seed.yaml")
	if err := os.WriteFile(path, []byte("version: 1\nclock: 2026-01-01T00:00:00Z\nprocessing_seconds: 5\n"), 0600); err != nil {
		t.Fatal(err)
	}
	seed := Must(path)
	if seed.Version != 1 || seed.ProcessingSeconds != 5 {
		t.Fatalf("unexpected seed: %+v", seed)
	}
}

func TestMustPanicsWithoutProfileSecrets(t *testing.T) {
	for _, contents := range []string{"account: secret-do-not-log", "version: 0\n"} {
		t.Run(contents, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "seed.yaml")
			if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if got := recover(); got != "profiles: initialization failed" {
					t.Fatalf("panic = %v", got)
				}
			}()
			Must(path)
			t.Fatal("Must did not panic")
		})
	}
}
