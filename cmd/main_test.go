package main

import (
	"encoding/json"
	"errors"
	"flag"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseStartHelp(t *testing.T) {
	if _, err := parseStart([]string{"--help"}); !errors.Is(err, flag.ErrHelp) {
		t.Fatalf("help error = %v", err)
	}
}

func TestParseStart(t *testing.T) {
	cfg, err := parseStart([]string{"--db", ":memory:", "--max-records", "42"})
	if err != nil || cfg.Database != ":memory:" || cfg.MaxRecords != 42 {
		t.Fatalf("parseStart = (%+v, %v)", cfg, err)
	}
}

// Keep the composition-root rule enforceable as more startup dependencies appear.
func TestMustArchitecture(t *testing.T) {
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, ".gen.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		var inspect func(ast.Node, bool, bool)
		inspect = func(root ast.Node, allowMust, allowPanic bool) {
			ast.Inspect(root, func(node ast.Node) bool {
				if literal, ok := node.(*ast.FuncLit); ok {
					inspect(literal.Body, false, false)
					return false
				}
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				name := ""
				switch target := call.Fun.(type) {
				case *ast.Ident:
					name = target.Name
				case *ast.SelectorExpr:
					name = target.Sel.Name
				}
				if name == "panic" && !allowPanic {
					t.Errorf("%s: panic outside a Must function", path)
				}
				if strings.HasPrefix(name, "Must") && !allowMust {
					t.Errorf("%s: %s called outside main", path, name)
				}
				return true
			})
		}
		for _, declaration := range file.Decls {
			if function, ok := declaration.(*ast.FuncDecl); ok {
				inspect(function.Body, file.Name.Name == "main" && function.Name.Name == "main", strings.HasPrefix(function.Name.Name, "Must"))
			} else {
				inspect(declaration, false, false)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestMainStartupFailures(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	for _, tc := range []struct {
		name         string
		args         []string
		panicMessage string
	}{
		{"config", []string{"start", "--addr", "secret-do-not-log"}, "config: initialization failed"},
		{"profile", []string{"start", "--seed", filepath.Join(t.TempDir(), "secret-do-not-log")}, "profiles: initialization failed"},
		{"database", []string{"start", "--db", "file::memory:?_journal_mode=secret-do-not-log"}, "db: initialization failed"},
		{"listener", []string{"start", "--addr", listener.Addr().String()}, "app: initialization failed"},
		{"start help", []string{"start", "--help"}, ""},
		{"reset help", []string{"reset", "--help"}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args, err := json.Marshal(tc.args)
			if err != nil {
				t.Fatal(err)
			}
			command := exec.Command(os.Args[0], "-test.run=^TestMainSubprocess$")
			command.Env = append(os.Environ(), "SOCIAL_MOCKERY_TEST_ARGS="+string(args), "OTEL_SDK_DISABLED=true")
			output, err := command.CombinedOutput()
			if tc.panicMessage == "" {
				if err != nil {
					t.Fatalf("help failed: %v\n%s", err, output)
				}
			} else if err == nil || !strings.Contains(string(output), "panic: "+tc.panicMessage) {
				t.Fatalf("startup failure = %v\n%s", err, output)
			}
			if strings.Contains(string(output), "secret-do-not-log") {
				t.Fatalf("startup output leaked configuration: %s", output)
			}
		})
	}
}

func TestMainSubprocess(t *testing.T) {
	serialized := os.Getenv("SOCIAL_MOCKERY_TEST_ARGS")
	if serialized == "" {
		return
	}
	var args []string
	if err := json.Unmarshal([]byte(serialized), &args); err != nil {
		t.Fatal(err)
	}
	os.Args = append([]string{"social-mockery"}, args...)
	main()
	os.Exit(0)
}
