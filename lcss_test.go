package lightningcss

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMinify(t *testing.T) {
	const input = `
		.foo {
			color: #ff0000;
			margin: 10px 20px 10px 20px;
		}
	`
	const want = `.foo{color:red;margin:10px 20px}`

	got, err := Minify([]byte(input))
	if err != nil {
		t.Fatalf("Minify: %v", err)
	}
	if string(got) != want {
		t.Fatalf("Minify\n\tgot: %q\n\twant: %q", got, want)
	}
}

// TestMinifyLenient confirms that lightningcss's lenient parsing recovers from
// malformed input without panicking the wasm guest.
func TestMinifyLenient(t *testing.T) {
	got, err := Minify([]byte(".foo { color: red;; ; } @media screen and { .x {} "))
	if err != nil {
		t.Fatalf("Minify: %v", err)
	}
	t.Logf("recovered output: %q", got)
}

func TestTargetsNestingLowering(t *testing.T) {
	res, err := Transform([]byte(`.a { .b { color: green; } }`), &Options{
		Minify:  true,
		Nesting: true,
		Targets: Targets{Chrome: Version(100, 0, 0)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(res.Code); got != `.a .b{color:green}` {
		t.Fatalf("got %q", got)
	}
}

func TestTargetsVendorPrefix(t *testing.T) {
	res, err := Transform([]byte(`.a { user-select: none }`), &Options{
		Minify:  true,
		Targets: Targets{Safari: Version(14, 0, 0), Chrome: Version(80, 0, 0)},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := string(res.Code)
	if !strings.Contains(got, "-webkit-user-select:none") {
		t.Fatalf("expected -webkit- prefix, got %q", got)
	}
}

func TestCSSModules(t *testing.T) {
	res, err := Transform([]byte(`.foo { color: red; }`), &Options{
		Minify:     true,
		Filename:   "style.module.css",
		CSSModules: &CSSModules{},
	})
	if err != nil {
		t.Fatal(err)
	}
	exp, ok := res.Exports["foo"]
	if !ok {
		t.Fatalf("missing export for .foo; exports=%v", res.Exports)
	}
	if exp.Name == "" || exp.Name == "foo" {
		t.Fatalf("expected scoped name, got %q", exp.Name)
	}
	if !strings.Contains(string(res.Code), "."+exp.Name+"{") {
		t.Fatalf("output %q does not use scoped name %q", res.Code, exp.Name)
	}
}

func TestAnalyzeDependencies(t *testing.T) {
	res, err := Transform([]byte(`.a { background: url(image.png) }`), &Options{
		Minify:              true,
		AnalyzeDependencies: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d: %+v", len(res.Dependencies), res.Dependencies)
	}
	dep := res.Dependencies[0]
	if dep.Type != "url" || dep.URL != "image.png" {
		t.Fatalf("unexpected dependency: %+v", dep)
	}
	if dep.Placeholder == "" || !strings.Contains(string(res.Code), dep.Placeholder) {
		t.Fatalf("placeholder %q not found in output %q", dep.Placeholder, res.Code)
	}
}

func TestSourceMap(t *testing.T) {
	res, err := Transform([]byte(".a {\n  color: red;\n}\n"), &Options{
		Filename:  "in.css",
		Minify:    true,
		SourceMap: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Map) == 0 {
		t.Fatal("expected a source map")
	}
	var sm struct {
		Version  int      `json:"version"`
		Mappings string   `json:"mappings"`
		Sources  []string `json:"sources"`
	}
	if err := json.Unmarshal(res.Map, &sm); err != nil {
		t.Fatalf("invalid source map JSON: %v", err)
	}
	if sm.Version != 3 || sm.Mappings == "" {
		t.Fatalf("unexpected source map: %+v", sm)
	}
}

func TestWarnings(t *testing.T) {
	res, err := Transform([]byte(`.a { color: red; } @import "b.css";`), &Options{
		ErrorRecovery: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Warnings) == 0 {
		t.Fatalf("expected a warning, got none (code=%q)", res.Code)
	}
	t.Logf("warnings: %v", res.Warnings)
}

func TestPseudoClasses(t *testing.T) {
	res, err := Transform([]byte(`.a:hover { color: red }`), &Options{
		Minify:        true,
		PseudoClasses: &PseudoClasses{Hover: "is-hover"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(res.Code); got != `.a.is-hover{color:red}` {
		t.Fatalf("got %q", got)
	}
}

func TestStyleAttribute(t *testing.T) {
	res, err := Transform([]byte(`color: red; margin: 10px 20px 10px 20px`), &Options{
		Minify:         true,
		StyleAttribute: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := string(res.Code); got != `color:red;margin:10px 20px` {
		t.Fatalf("got %q", got)
	}
}

func TestUnusedSymbols(t *testing.T) {
	res, err := Transform([]byte(`.used { color: red } .unused { color: blue }`), &Options{
		Minify:        true,
		UnusedSymbols: []string{"unused"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := string(res.Code)
	if strings.Contains(got, "unused") || !strings.Contains(got, ".used") {
		t.Fatalf("unused symbol not removed: %q", got)
	}
}

func TestConcurrent(t *testing.T) {
	const css = `.a { color: #ff0000 }`
	done := make(chan string, 64)
	for range cap(done) {
		go func() {
			out, err := Minify([]byte(css))
			if err != nil {
				done <- "err: " + err.Error()
				return
			}
			done <- string(out)
		}()
	}
	for range cap(done) {
		if got := <-done; got != `.a{color:red}` {
			t.Fatalf("incorrect %q", got)
		}
	}
}
