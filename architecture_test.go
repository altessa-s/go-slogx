// Copyright 2021-2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by the MIT license that can be found in
// the LICENSE file.

package slogx_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

const module = "github.com/altessa-s/go-slogx"

// goAtlasCore is the foundation module. Adapters and the test kit may import it.
const goAtlasCore = "github.com/altessa-s/go-atlas-core/..."

// layers defines the allowed dependencies. A pattern ending in "/..." matches
// a path and everything under it. A package missing from this table fails the test.
var layers = map[string][]string{
	".":                    nil,
	"mask":                 nil,
	"internal/handlerbase": nil,
	"internal/pathcache":   nil,
	"internal/attrwalk":    {module + "/mask"},
	"handler/leveled":      {module, module + "/internal/handlerbase"},
	"handler/prefixed":     {module, module + "/internal/handlerbase"},
	"handler/masking": {
		module,
		module + "/mask",
		module + "/internal/handlerbase",
		module + "/internal/attrwalk",
		module + "/internal/pathcache",
	},
	"console":   {module, module + "/internal/handlerbase", goAtlasCore},
	"bufwriter": nil,
	"setup": {
		module,
		module + "/mask",
		module + "/handler/leveled",
		module + "/handler/prefixed",
		module + "/handler/masking",
		module + "/console",
		module + "/bufwriter",
		goAtlasCore,
		"github.com/mattn/go-isatty",
	},
	"slogxtest": {module, module + "/internal/handlerbase", "github.com/stretchr/testify/...", goAtlasCore},
}

// devtoolsLayers allowlists the blank imports of devtools/tools.go.
var devtoolsLayers = map[string][]string{
	".": {
		"github.com/altessa-s/go-copyright-checker",
		"github.com/altessa-s/go-optgen/cmd/optgen",
		"github.com/daixiang0/gci",
		"github.com/golangci/golangci-lint/v2/cmd/golangci-lint",
		"github.com/mibk/dupl",
		"github.com/securego/gosec/v2/cmd/gosec",
		"golang.org/x/perf/cmd/benchstat",
		"golang.org/x/vuln/cmd/govulncheck",
	},
}

// allowedRootRequires is the direct-require allowlist of the root go.mod.
var allowedRootRequires = map[string]struct{}{
	"github.com/altessa-s/go-atlas-core": {},
	"github.com/mattn/go-isatty":         {},
	"github.com/stretchr/testify":        {},
}

// TestArchitecture enforces the layer table on every module.
func TestArchitecture(t *testing.T) {
	t.Parallel()

	for _, root := range moduleRoots(t) {
		switch root {
		case ".":
			checkLayers(t, root, layers, "slogxtest")
		case "devtools":
			checkLayers(t, root, devtoolsLayers, "")
		default:
			t.Errorf("module %s has no architecture table", root)
		}
	}
}

// TestNoMonolithImports rejects the go-atlas monolith, sibling network modules,
// and fatih/color in every Go file of the main module, tests included.
func TestNoMonolithImports(t *testing.T) {
	t.Parallel()

	for _, root := range moduleRoots(t) {
		for _, file := range goFiles(t, root, true) {
			for _, imp := range fileImports(t, file.path) {
				if root == "." && strings.HasPrefix(imp, "github.com/altessa-s/") &&
					!modulePrefix(imp, module) && !modulePrefix(imp, "github.com/altessa-s/go-atlas-core") {
					t.Errorf("%s imports unlisted sibling %s", file.rel, imp)
				}

				if root == "." && colorImport(imp) {
					t.Errorf("%s imports %s", file.rel, imp)
				}

				if monolithOrSibling(imp) {
					t.Errorf("%s imports %s", file.rel, imp)
				}
			}
		}
	}
}

// TestGoModPurity checks require allowlists and forbids replace directives.
func TestGoModPurity(t *testing.T) {
	t.Parallel()
	rootMods := readGoMod(t, "go.mod")
	require.False(t, rootMods.hasReplace, "root go.mod must not contain a replace directive")

	for _, req := range rootMods.direct {
		_, ok := allowedRootRequires[req]
		require.True(t, ok, "root go.mod direct require %s is not allowed", req)
	}

	for _, req := range rootMods.all {
		require.False(t, monolithOrSiblingModule(req), "root go.mod requires %s", req)
		require.False(t, colorModule(req), "root go.mod requires %s", req)
		require.False(t, otherAltessa(req), "root go.mod requires sibling module %s", req)
	}

	devMods := readGoMod(t, filepath.Join("devtools", "go.mod"))
	require.False(t, devMods.hasReplace, "devtools go.mod must not contain a replace directive")

	for _, req := range devMods.all {
		require.False(t, monolithOrSiblingModule(req), "devtools go.mod requires %s", req)
	}
}

// TestProductionConstraints bans exported mutable state, init, and hand-written With functions.
func TestProductionConstraints(t *testing.T) {
	t.Parallel()

	for _, root := range moduleRoots(t) {
		for _, file := range goFiles(t, root, true) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, file.path, nil, parser.ParseComments)
			require.NoError(t, err)
			ast.Inspect(f, func(n ast.Node) bool {
				switch node := n.(type) {
				case *ast.GenDecl:
					if node.Tok != token.VAR || strings.HasSuffix(file.rel, "_test.go") {
						return true
					}

					for _, spec := range node.Specs {
						vs, ok := spec.(*ast.ValueSpec)
						if !ok {
							continue
						}

						for _, name := range vs.Names {
							if name.IsExported() && !errorSentinel(vs, f) {
								t.Errorf("%s exports mutable %s", file.rel, name.Name)
							}
						}
					}
				case *ast.FuncDecl:
					if node.Recv == nil && node.Name.Name == "init" {
						t.Errorf("%s declares init", file.rel)
					}

					if handwrittenWith(node.Name.Name) && !ast.IsGenerated(f) &&
						!(node.Recv != nil && (node.Name.Name == "WithAttrs" || node.Name.Name == "WithGroup")) {
						t.Errorf("%s declares hand-written %s", file.rel, node.Name.Name)
					}
				}

				return true
			})
		}
	}
}

// TestPackageDocs requires doc.go and README.md beside every package.
func TestPackageDocs(t *testing.T) {
	t.Parallel()

	for _, root := range moduleRoots(t) {
		seen := map[string]struct{}{}

		for _, file := range goFiles(t, root, false) {
			dir := path.Dir(file.rel)
			if _, ok := seen[dir]; ok {
				continue
			}

			seen[dir] = struct{}{}

			base := root
			if dir != "." {
				base = filepath.Join(root, filepath.FromSlash(dir))
			}

			_, err := os.Stat(filepath.Join(base, "doc.go"))
			require.NoError(t, err, "%s/%s missing doc.go", root, dir)
			_, err = os.Stat(filepath.Join(base, "README.md"))
			require.NoError(t, err, "%s/%s missing README.md", root, dir)
		}
	}
}

func checkLayers(t *testing.T, root string, table map[string][]string, testingPkg string) {
	t.Helper()

	pkgs := productionImports(t, root)
	for dir := range table {
		_, found := pkgs[dir]
		require.True(t, found || strings.HasSuffix(dir, "*"), "layer package %s was not walked under %s", dir, root)
	}

	for dir, imports := range pkgs {
		allowed, ok := layerOf(table, dir)
		require.True(t, ok, "package %s is in no layer: add it to the layers table", dir)

		for _, imp := range imports {
			if stdlib(imp) {
				if imp == "testing" || strings.HasPrefix(imp, "testing/") {
					require.Equal(t, testingPkg, dir, "%s imports %s", dir, imp)
				}

				continue
			}

			require.True(t, slices.ContainsFunc(allowed, func(p string) bool { return matches(p, imp) }),
				"%s must not import %s (allowed: %v)", dir, imp, allowed)

			if strings.HasPrefix(dir, "handler/") {
				require.False(t, strings.HasPrefix(imp, module+"/handler/"), "%s imports sibling %s", dir, imp)
			}
		}
	}
}

func layerOf(table map[string][]string, dir string) ([]string, bool) {
	if allowed, ok := table[dir]; ok {
		return allowed, true
	}

	parent, _ := path.Split(dir)
	allowed, ok := table[parent+"*"]

	return allowed, ok
}

func matches(pattern, imp string) bool {
	if prefix, ok := strings.CutSuffix(pattern, "/..."); ok {
		return imp == prefix || strings.HasPrefix(imp, prefix+"/")
	}

	return imp == pattern
}

func stdlib(imp string) bool {
	first, _, _ := strings.Cut(imp, "/")
	return !strings.Contains(first, ".")
}

type goFile struct {
	path string
	rel  string
}

func productionImports(t *testing.T, root string) map[string][]string {
	t.Helper()

	pkgs := map[string][]string{}

	for _, file := range goFiles(t, root, false) {
		dir := path.Dir(file.rel)
		for _, imp := range fileImports(t, file.path) {
			if !slices.Contains(pkgs[dir], imp) {
				pkgs[dir] = append(pkgs[dir], imp)
			}
		}

		if _, ok := pkgs[dir]; !ok {
			pkgs[dir] = nil
		}
	}

	return pkgs
}

func goFiles(t *testing.T, root string, includeTests bool) []goFile {
	t.Helper()

	absRoot, err := filepath.Abs(root)
	require.NoError(t, err)

	var out []goFile

	fsetErr := filepath.WalkDir(absRoot, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, relErr := filepath.Rel(absRoot, p)
		if relErr != nil {
			return relErr
		}

		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			name := d.Name()
			if rel != "." && (strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || name == "testdata") {
				return filepath.SkipDir
			}

			if rel == "devtools" && root == "." {
				return filepath.SkipDir
			}

			if rel != "." {
				if _, statErr := os.Stat(filepath.Join(p, "go.mod")); statErr == nil {
					return filepath.SkipDir
				}
			}

			return nil
		}

		if !strings.HasSuffix(rel, ".go") {
			return nil
		}

		if !includeTests && strings.HasSuffix(rel, "_test.go") {
			return nil
		}

		out = append(out, goFile{path: p, rel: rel})

		return nil
	})
	require.NoError(t, fsetErr)

	return out
}

func fileImports(t *testing.T, path string) []string {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
	require.NoError(t, err)

	out := make([]string, 0, len(f.Imports))
	for _, spec := range f.Imports {
		imp, err := strconv.Unquote(spec.Path.Value)
		require.NoError(t, err)

		out = append(out, imp)
	}

	return out
}

func handwrittenWith(name string) bool {
	rest, ok := strings.CutPrefix(name, "With")
	if !ok || rest == "" {
		return false
	}

	r, _ := utf8.DecodeRuneInString(rest)

	return unicode.IsUpper(r)
}

func monolithOrSibling(imp string) bool {
	return modulePrefix(imp, "github.com/altessa-s/go-atlas") ||
		modulePrefix(imp, "github.com/altessa-s/go-atlas-net") ||
		modulePrefix(imp, "github.com/altessa-s/go-tlsutils")
}

func monolithOrSiblingModule(mod string) bool {
	return monolithOrSibling(mod)
}

func colorImport(imp string) bool {
	return modulePrefix(imp, "github.com/fatih/color")
}

func colorModule(mod string) bool {
	return colorImport(mod)
}

func modulePrefix(imp, base string) bool {
	return imp == base || strings.HasPrefix(imp, base+"/")
}

func otherAltessa(mod string) bool {
	const prefix = "github.com/altessa-s/"

	rest, ok := strings.CutPrefix(mod, prefix)
	if !ok {
		return false
	}

	name, _, _ := strings.Cut(rest, "/")
	switch name {
	case "go-atlas-core", "go-slogx":
		return false
	default:
		return true
	}
}

type goMod struct {
	direct     []string
	all        []string
	hasReplace bool
}

func readGoMod(t *testing.T, path string) goMod {
	t.Helper()

	body, err := os.ReadFile(path)
	require.NoError(t, err)

	var parsed goMod

	inRequire := false

	for line := range strings.SplitSeq(string(body), "\n") {
		code, comment, _ := strings.Cut(line, "//")

		fields := strings.Fields(code)
		if len(fields) == 0 {
			continue
		}

		if fields[0] == "replace" {
			parsed.hasReplace = true
		}

		indirect := strings.Contains(comment, "indirect")

		if fields[0] == "require" {
			if len(fields) >= 3 {
				parsed.add(fields[1], indirect)
			}

			if slices.Contains(fields, "(") {
				inRequire = true
			}

			continue
		}

		if inRequire {
			if fields[0] == ")" {
				inRequire = false
				continue
			}

			if len(fields) >= 2 {
				parsed.add(fields[0], indirect)
			}
		}
	}

	return parsed
}

func (m *goMod) add(mod string, indirect bool) {
	m.all = append(m.all, mod)
	if !indirect {
		m.direct = append(m.direct, mod)
	}
}

func moduleRoots(t *testing.T) []string {
	t.Helper()

	var roots []string

	err := filepath.WalkDir(".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && p != "." && (strings.HasPrefix(d.Name(), ".") || d.Name() == "testdata" || d.Name() == "bin") {
			return filepath.SkipDir
		}

		if !d.IsDir() && d.Name() == "go.mod" {
			roots = append(roots, filepath.Dir(p))
		}

		return nil
	})
	require.NoError(t, err)
	slices.Sort(roots)

	return roots
}

// TestRemovedAPIs guards the intentional removals behind D7, D11 and D20.
func TestRemovedAPIs(t *testing.T) {
	t.Parallel()

	for _, dir := range []string{"handler/buffered", "handler/fanout", "handler/multi"} {
		_, err := os.Stat(dir)
		require.True(t, os.IsNotExist(err), "removed package exists: %s", dir)
	}

	removed := map[string]bool{"Hash": true, "HashMask": true, "S3URLMask": true, "SmartMask": true, "CachedPartialMask": true, "PrecomputedMasks": true, "GlobalLevel": true, "Shutdown": true}

	for _, file := range goFiles(t, ".", false) {
		dir := path.Dir(file.rel)
		if dir != "." && dir != "mask" {
			continue
		}

		syntax, err := parser.ParseFile(token.NewFileSet(), file.path, nil, 0)
		require.NoError(t, err)

		for _, declaration := range syntax.Decls {
			if fn, ok := declaration.(*ast.FuncDecl); ok && fn.Recv == nil {
				require.False(t, removed[fn.Name.Name], "removed API: %s", fn.Name.Name)
			}

			if group, ok := declaration.(*ast.GenDecl); ok {
				for _, spec := range group.Specs {
					switch node := spec.(type) {
					case *ast.TypeSpec:
						require.False(t, removed[node.Name.Name])
					case *ast.ValueSpec:
						for _, name := range node.Names {
							require.False(t, removed[name.Name])
						}
					}
				}
			}
		}
	}
}

// TestH1BlackBoxPackages requires explicit filenames for internal test access.
func TestH1BlackBoxPackages(t *testing.T) {
	t.Parallel()

	for _, root := range moduleRoots(t) {
		for _, file := range goFiles(t, root, true) {
			if !strings.HasSuffix(file.rel, "_test.go") || strings.HasSuffix(file.rel, "_internal_test.go") {
				continue
			}

			f, err := parser.ParseFile(token.NewFileSet(), file.path, nil, parser.PackageClauseOnly)
			require.NoError(t, err)
			require.True(t, strings.HasSuffix(f.Name.Name, "_test"), "%s must use an external test package", file.rel)
		}
	}
}

func errorSentinel(v *ast.ValueSpec, file *ast.File) bool {
	if len(v.Names) != 1 || len(v.Values) != 1 {
		return false
	}

	name := v.Names[0].Name
	if len(name) <= len("Err") || !strings.HasPrefix(name, "Err") || name[3] < 'A' || name[3] > 'Z' {
		return false
	}

	call, ok := v.Values[0].(*ast.CallExpr)
	if !ok {
		return false
	}

	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	pkg, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}

	imported := ""

	for _, imp := range file.Imports {
		value, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}

		alias := path.Base(value)
		if imp.Name != nil {
			alias = imp.Name.Name
		}

		if alias == pkg.Name {
			imported = value
		}
	}

	if imported == "errors" && sel.Sel.Name == "New" {
		return len(call.Args) == 1
	}

	if imported != "fmt" || sel.Sel.Name != "Errorf" || len(call.Args) == 0 {
		return false
	}

	literal, ok := call.Args[0].(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return false
	}

	format, err := strconv.Unquote(literal.Value)
	// Reject all wrapping forms, including indexed and width-qualified verbs.
	return err == nil && !wrapsError(format)
}

func wrapsError(format string) bool {
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}

		i++
		if i < len(format) && format[i] == '%' {
			continue
		}

		for ; i < len(format); i++ {
			c := format[i]
			if c == 'w' {
				return true
			}

			if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' {
				break
			}
		}
	}

	return false
}

func TestH1ErrorSentinelPolicy(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		decl    string
		allowed bool
	}{
		{`var ErrValid = errors.New("failure")`, true},
		{`var ErrWarning = fmt.Errorf("warning: %s", "failure")`, true},
		{`var ErrWrapped = fmt.Errorf("%[1]w", err)`, false},
		{`var ErrWrapped = fmt.Errorf("%10w", err)`, false},
		{`var ErrEscaped = fmt.Errorf("%%w")`, true},
		{`var Error = errors.New("failure")`, false},
		{`var ErrMap = map[string]string{}`, false},
		{`var ErrAlias = another`, false},
		{`var ErrOne, ErrTwo = errors.New("one"), errors.New("two")`, false},
		{`var ErrDynamic = fmt.Errorf(format)`, false},
	} {
		f, err := parser.ParseFile(token.NewFileSet(), "policy.go", "package policy; import (\"errors\"; \"fmt\");"+tc.decl, 0)
		require.NoError(t, err)

		declaration, ok := f.Decls[1].(*ast.GenDecl)
		require.True(t, ok)
		v, ok := declaration.Specs[0].(*ast.ValueSpec)
		require.True(t, ok)
		require.Equal(t, tc.allowed, errorSentinel(v, f), tc.decl)
	}
}
