package etag

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// exportedPackageNames parses every non-test Go file in dir and returns the
// names of all exported package-level declarations (functions without
// receivers, types, and constants).
func exportedPackageNames(t *testing.T, dir string) map[string]bool {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %s: %v", dir, err)
	}

	names := make(map[string]bool)

	for _, entry := range entries {
		fileName := entry.Name()

		if entry.IsDir() || !strings.HasSuffix(fileName, ".go") || strings.HasSuffix(fileName, "_test.go") {
			continue
		}

		fset := token.NewFileSet()

		file, err := parser.ParseFile(fset, filepath.Join(dir, fileName), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", fileName, err)
		}

		for _, decl := range file.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && d.Name.IsExported() {
					names[d.Name.Name] = true
				}

			case *ast.GenDecl:
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if s.Name.IsExported() {
							names[s.Name.Name] = true
						}

					case *ast.ValueSpec:
						for _, id := range s.Names {
							if id.IsExported() {
								names[id.Name] = true
							}
						}
					}
				}
			}
		}
	}

	return names
}

// TestServerReexportsFullEntityTagSurface pins the server package's
// re-export contract over the entitytag module: every exported
// package-level name in entitytag must also exist in this package, so a
// new entitytag export cannot silently miss the historical etag.* surface
// (the same guarantee deprecated_test.go gives the root shim).
func TestServerReexportsFullEntityTagSurface(t *testing.T) {
	t.Parallel()

	const entitytagDir = "../entitytag"

	if _, err := os.Stat(entitytagDir); err != nil {
		t.Skipf("entitytag source not adjacent to the server module: %v", err)
	}

	entitytagNames := exportedPackageNames(t, entitytagDir)
	serverNames := exportedPackageNames(t, ".")

	for name := range entitytagNames {
		if !serverNames[name] {
			t.Errorf("entitytag export %s is missing from the server re-export surface", name)
		}
	}
}

// TestParseETagListReexport exercises the re-exported ParseETagList wrapper
// end to end, covering the shim function that no server-package test calls.
func TestParseETagListReexport(t *testing.T) {
	t.Parallel()

	tags := ParseETagList(`"a", W/"b"`)

	if len(tags) != 2 {
		t.Fatalf("ParseETagList returned %d tags, want 2", len(tags))
	}

	if got := tags[1].String(); got != `W/"b"` {
		t.Errorf("second tag String() = %s, want W/%q", got, `"b"`)
	}
}
