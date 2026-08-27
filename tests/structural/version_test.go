package structural_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// devSentinel is what an unstamped build reports. It is the one three-part
// version string allowed to sit in the source.
const devSentinel = "0.0.0-dev"

var versionLiteral = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// readVersion returns the contents of VERSION, the single place a real version
// string is written down.
func readVersion(t *testing.T, root string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "VERSION"))
	if err != nil {
		t.Fatalf("read VERSION: %v", err)
	}
	version := strings.TrimSpace(string(raw))
	if !versionLiteral.MatchString(version) {
		t.Fatalf("VERSION does not hold a three-part version string: %q", version)
	}
	return version
}

// TestVersionIsAVarNotAConst guards the trap that made this rule necessary. The
// build sets the version with -ldflags "-X", which reaches a var and silently
// does nothing against a const: the binary would then ship the sentinel with no
// error anywhere to read.
func TestVersionIsAVarNotAConst(t *testing.T) {
	root, err := findProjectRoot()
	if err != nil {
		t.Fatalf("cannot find project root: %v", err)
	}

	path := filepath.Join(root, "internal", "version", "version.go")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	var found bool
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for _, name := range value.Names {
				if name.Name != "Version" {
					continue
				}
				found = true
				if gen.Tok == token.CONST {
					t.Error("internal/version.Version is a const: -ldflags -X cannot reach it, " +
						"so a build would ship the sentinel with no error to read. Declare it as a var.")
				}
			}
		}
	}
	if !found {
		t.Fatal("internal/version.Version is not declared: the build has nothing to stamp")
	}
}

// TestNoVersionLiteralInGoSource holds VERSION as the single source of truth on
// the Go side. The sentinel is the only three-part string allowed through.
func TestNoVersionLiteralInGoSource(t *testing.T) {
	root, err := findProjectRoot()
	if err != nil {
		t.Fatalf("cannot find project root: %v", err)
	}

	files := collectGoFiles(t, filepath.Join(root, "internal"))
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("read root: %v", err)
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".go") {
			files = append(files, filepath.Join(root, entry.Name()))
		}
	}

	for _, file := range files {
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", file, err)
		}
		rel, _ := filepath.Rel(root, file)
		ast.Inspect(parsed, func(node ast.Node) bool {
			lit, ok := node.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			value := strings.Trim(lit.Value, "`\"")
			if value == devSentinel || !versionLiteral.MatchString(value) {
				return true
			}
			t.Errorf("%s:%d: version literal %q in Go source; VERSION is the only place one may be written",
				rel, fset.Position(lit.Pos()).Line, value)
			return true
		})
	}
}

// TestStaticFilesAreStamped catches the other half of the drift: a VERSION that
// was bumped while the files nothing can stamp at render time kept the old
// figure. Run ./stamp_version.ps1 to repair it.
func TestStaticFilesAreStamped(t *testing.T) {
	root, err := findProjectRoot()
	if err != nil {
		t.Fatalf("cannot find project root: %v", err)
	}
	version := readVersion(t, root)

	stamps := []struct {
		path    string
		pattern *regexp.Regexp
		want    string
	}{
		{
			path:    filepath.Join("build", "windows", "info.json"),
			pattern: regexp.MustCompile(`"file_version"\s*:\s*"([^"]*)"`),
			want:    version + ".0",
		},
		{
			path:    filepath.Join("build", "windows", "info.json"),
			pattern: regexp.MustCompile(`"ProductVersion"\s*:\s*"([^"]*)"`),
			want:    version,
		},
	}

	pages, err := filepath.Glob(filepath.Join(root, "docs", "*.html"))
	if err != nil {
		t.Fatalf("glob docs: %v", err)
	}
	if len(pages) == 0 {
		t.Fatal("no site pages found: the stamp guard would pass by covering nothing")
	}
	token := regexp.MustCompile(`<!--VERSION-->([^<]*)<!--/VERSION-->`)
	for _, page := range pages {
		rel, _ := filepath.Rel(root, page)
		stamps = append(stamps, struct {
			path    string
			pattern *regexp.Regexp
			want    string
		}{path: rel, pattern: token, want: version})
	}

	for _, stamp := range stamps {
		raw, err := os.ReadFile(filepath.Join(root, stamp.path))
		if err != nil {
			t.Fatalf("read %s: %v", stamp.path, err)
		}
		match := stamp.pattern.FindStringSubmatch(string(raw))
		if match == nil {
			t.Errorf("%s: no %s stamp found; the version cannot reach this file", stamp.path, stamp.pattern)
			continue
		}
		if match[1] != stamp.want {
			t.Errorf("%s: stamped %q, VERSION says %q; run ./stamp_version.ps1",
				stamp.path, match[1], stamp.want)
		}
	}
}
