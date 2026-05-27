package structural_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const modulePrefix = "github.com/oernster/locus/internal/"

// Layer import rules: a layer must NOT import the listed forbidden layers.
var rules = []struct {
	layer     string
	forbidden []string
}{
	{
		layer:     "domain",
		forbidden: []string{"application", "infrastructure"},
	},
	{
		layer:     "application",
		forbidden: []string{"infrastructure"},
	},
}

func TestLayerBoundaries(t *testing.T) {
	root, err := findProjectRoot()
	if err != nil {
		t.Fatalf("cannot find project root: %v", err)
	}

	internalDir := filepath.Join(root, "internal")

	for _, rule := range rules {
		rule := rule
		t.Run("layer_"+rule.layer, func(t *testing.T) {
			layerDir := filepath.Join(internalDir, rule.layer)
			goFiles := collectGoFiles(t, layerDir)

			for _, file := range goFiles {
				imports := parseImports(t, file)
				for _, imp := range imports {
					for _, forbidden := range rule.forbidden {
						if strings.Contains(imp, modulePrefix+forbidden) {
							rel, _ := filepath.Rel(root, file)
							t.Errorf("boundary violation: %s imports %s (layer %q must not import layer %q)",
								rel, imp, rule.layer, forbidden)
						}
					}
				}
			}
		})
	}
}

func collectGoFiles(t *testing.T, dir string) []string {
	t.Helper()
	var files []string
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return files
	}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", dir, err)
	}
	return files
}

func parseImports(t *testing.T, file string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	var imports []string
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		imports = append(imports, path)
	}
	return imports
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", os.ErrNotExist
}
