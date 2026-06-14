// Package lightningcss provides Go bindings for [LightningCSS].
//
// [LightningCSS]: https://github.com/parcel-bundler/lightningcss
package lightningcss

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	lightningcss_wasm "github.com/pgaskin/go-lightningcss/internal"
)

//go:generate docker build --platform amd64 --progress plain --output . src

// Options configures a CSS transformation. The zero value is valid and parses,
// minimally normalizes, and pretty-prints the input.
type Options struct {
	// Filename is the source file name, used in error messages, source maps,
	// and CSS modules name hashing.
	Filename string

	// Minify removes whitespace and other redundancy from the output.
	// Regardless of this setting, lightningcss always normalizes values and,
	// when Targets is set, lowers modern syntax and adds vendor prefixes.
	Minify bool

	// Targets sets the minimum browser versions to support. See [Targets].
	Targets Targets

	// ErrorRecovery continues past most parse and minify errors, emitting them
	// as warnings (see [Result.Warnings]) instead of failing.
	ErrorRecovery bool

	// Nesting enables parsing of CSS nesting.
	Nesting bool

	// CustomMedia enables parsing of @custom-media rules.
	CustomMedia bool

	// DeepSelectorCombinator enables parsing of the non-standard >>> and /deep/
	// combinators.
	DeepSelectorCombinator bool

	// UnusedSymbols is a set of class names, ids, @keyframes names, CSS
	// variables, and other identifiers to remove from the output.
	UnusedSymbols []string

	// CSSModules, if non-nil, enables CSS modules compilation.
	CSSModules *CSSModules

	// PseudoClasses, if non-nil, substitutes user-action pseudo-classes with
	// regular classes.
	PseudoClasses *PseudoClasses

	// AnalyzeDependencies replaces @import and url() references with
	// placeholders and reports them in [Result.Dependencies].
	AnalyzeDependencies bool

	// RemoveImports removes @import rules when AnalyzeDependencies is set (they
	// are expected to be handled by the caller's bundler).
	RemoveImports bool

	// SourceMap generates a source map, returned in [Result.Map].
	SourceMap bool

	// InputSourceMap is an existing source map (JSON) for the input that the
	// generated source map should extend. Only used when SourceMap is set.
	InputSourceMap []byte

	// ProjectRoot is the root directory used to make source map paths relative.
	ProjectRoot string

	// StyleAttribute parses the input as the value of an inline style attribute
	// (a declaration list) rather than a full stylesheet.
	StyleAttribute bool
}

// Targets specifies the minimum browser versions to compile CSS syntax for.
// Each value is an encoded version number: (major << 16) | (minor << 8) |
// patch. A zero value means the browser is not targeted. Use [Version] to
// construct an encoded value.
//
// When any target is set, lightningcss compiles modern syntax (nesting, custom
// media, color functions, logical properties, etc.) and adds vendor prefixes as
// needed so the output works in the targeted browsers.
type Targets struct {
	Android   uint32
	Chrome    uint32
	Edge      uint32
	Firefox   uint32
	IE        uint32
	IOSSafari uint32
	Opera     uint32
	Safari    uint32
	Samsung   uint32
}

// Version encodes a browser version as expected by [Targets].
func Version(major, minor, patch uint8) uint32 {
	return uint32(major)<<16 | uint32(minor)<<8 | uint32(patch)
}

// IsZero reports whether no targets are set.
func (t Targets) IsZero() bool {
	return t == Targets{}
}

// CSSModules configures CSS modules compilation. Class names and other
// identifiers are scoped, and the mapping is returned in [Result.Exports].
type CSSModules struct {
	// Pattern is the naming pattern for scoped names, e.g. "[hash]_[local]" or
	// "[path][name]_[local]". If empty, the default pattern is used.
	Pattern string
	// DashedIdents scopes variables prefixed with "--" as well.
	DashedIdents bool
	// Pure requires class/id selectors in every rule (CSS Modules "pure" mode).
	Pure bool
	// Animation, Grid, CustomIdents, and Container scope the respective kinds
	// of identifiers. They default to true. Set a pointer to false to disable
	// one.
	Animation    *bool
	Grid         *bool
	CustomIdents *bool
	Container    *bool
}

// PseudoClasses replaces user-action pseudo-classes with regular classes so
// that states can be forced for testing or screenshots. Each field, if
// non-empty, is the class name to substitute for that pseudo-class.
type PseudoClasses struct {
	Hover        string
	Active       string
	Focus        string
	FocusVisible string
	FocusWithin  string
}

// Dependency is an `@import` or `url()` reference discovered when
// [Options.AnalyzeDependencies] is enabled. The original reference is replaced
// in the output with Placeholder, which callers can substitute after resolving.
type Dependency struct {
	// Type is "import" or "url".
	Type string
	// URL is the referenced URL.
	URL string
	// Placeholder is the unique string substituted into the output for this
	// dependency.
	Placeholder string
	// Supports is the value of the @import supports() condition, if any (import
	// dependencies only).
	Supports string
	// Media is the media query of the @import, if any (import dependencies
	// only).
	Media string
	// Loc is the source location of the dependency.
	Loc SourceRange
}

// SourceRange is a span in the source file.
type SourceRange struct {
	FilePath string
	Start    Location
	End      Location
}

// Location is a 1-based line/column position.
type Location struct {
	Line   uint32
	Column uint32
}

// CSSModuleExport is a single exported identifier from a CSS module.
type CSSModuleExport struct {
	// Name is the compiled (scoped) name.
	Name string
	// Composes lists other names this export composes, in order.
	Composes []CSSModuleReference
	// IsReferenced reports whether the export is referenced within the module.
	IsReferenced bool
}

// CSSModuleReference is a reference from one CSS module export to another.
type CSSModuleReference struct {
	// Type is "local", "global", or "dependency".
	Type string
	// Name is the referenced name.
	Name string
	// Specifier is the module specifier the name is referenced from (dependency
	// references only).
	Specifier string
}

// Result is the output of a transformation.
type Result struct {
	// Code is the transformed CSS.
	Code []byte

	// Map is the generated source map (JSON), or nil if Options.SourceMap was
	// not set.
	Map []byte

	// Exports maps the original to the compiled identifiers when CSS modules
	// are enabled.
	Exports map[string]CSSModuleExport

	// References maps placeholders to CSS module references when CSS modules
	// are enabled.
	References map[string]CSSModuleReference

	// Dependencies lists @import and url() references when
	// Options.AnalyzeDependencies is set.
	Dependencies []Dependency

	// Warnings lists non-fatal diagnostics produced during parsing and
	// minifying.
	Warnings []string
}

// Minify minifies css leniently. It is identical to calling [Transform] with
// [Options.Minify].
func Minify(css []byte) ([]byte, error) {
	res, err := Transform(css, &Options{Minify: true})
	if err != nil {
		return nil, err
	}
	return res.Code, nil
}

// Transform parses, transforms, and serializes css.
func Transform(css []byte, opts *Options) (*Result, error) {
	if opts == nil {
		opts = &Options{}
	}
	req, err := json.Marshal(request(opts))
	if err != nil {
		return nil, err
	}

	in := newInstance()
	var res *Result
	err = in.call(func() error {
		srcPtr, err := in.writeBytes(css)
		if err != nil {
			return err
		}
		reqPtr, err := in.writeBytes(req)
		if err != nil {
			return err
		}
		outPtr, err := in.alloc(8 * 4) // [8]uint32
		if err != nil {
			return err
		}
		in.mod.Xlcss_transform(srcPtr, int32(len(css)), reqPtr, int32(len(req)), outPtr)
		res, err = in.readResult(outPtr)
		return err
	})
	return res, err
}

func request(o *Options) *lightningcss_wasm.Request {
	r := &lightningcss_wasm.Request{
		Filename:               o.Filename,
		Minify:                 o.Minify,
		SourceMap:              o.SourceMap,
		ProjectRoot:            o.ProjectRoot,
		ErrorRecovery:          o.ErrorRecovery,
		Nesting:                o.Nesting,
		CustomMedia:            o.CustomMedia,
		DeepSelectorCombinator: o.DeepSelectorCombinator,
		UnusedSymbols:          o.UnusedSymbols,
		AnalyzeDependencies:    o.AnalyzeDependencies,
		RemoveImports:          o.RemoveImports,
		StyleAttribute:         o.StyleAttribute,
	}
	if len(o.InputSourceMap) != 0 {
		r.InputSourceMap = string(o.InputSourceMap)
	}
	if !o.Targets.IsZero() {
		r.Targets = &lightningcss_wasm.TargetsReq{
			Android:   o.Targets.Android,
			Chrome:    o.Targets.Chrome,
			Edge:      o.Targets.Edge,
			Firefox:   o.Targets.Firefox,
			IE:        o.Targets.IE,
			IOSSafari: o.Targets.IOSSafari,
			Opera:     o.Targets.Opera,
			Safari:    o.Targets.Safari,
			Samsung:   o.Targets.Samsung,
		}
	}
	if cm := o.CSSModules; cm != nil {
		r.CSSModules = &lightningcss_wasm.CSSModulesReq{
			Pattern:      cm.Pattern,
			DashedIdents: cm.DashedIdents,
			Pure:         cm.Pure,
			Animation:    cm.Animation,
			Grid:         cm.Grid,
			CustomIdents: cm.CustomIdents,
			Container:    cm.Container,
		}
	}
	if pc := o.PseudoClasses; pc != nil {
		r.PseudoClasses = &lightningcss_wasm.PseudoClassesReq{
			Hover:        pc.Hover,
			Active:       pc.Active,
			Focus:        pc.Focus,
			FocusVisible: pc.FocusVisible,
			FocusWithin:  pc.FocusWithin,
		}
	}
	return r
}

func result(m *lightningcss_wasm.Meta) *Result {
	r := &Result{Warnings: m.Warnings}
	r.Exports = mapValues(m.Exports, cssModuleExport)
	r.References = mapValues(m.References, func(c lightningcss_wasm.CSSModuleReference) CSSModuleReference {
		return CSSModuleReference(c)
	})
	if len(m.Dependencies) != 0 {
		r.Dependencies = make([]Dependency, len(m.Dependencies))
		for i, d := range m.Dependencies {
			r.Dependencies[i] = Dependency{
				Type:        d.Type,
				URL:         d.URL,
				Placeholder: d.Placeholder,
				Supports:    d.Supports,
				Media:       d.Media,
				Loc: SourceRange{
					FilePath: d.Loc.FilePath,
					Start:    Location{Line: d.Loc.Start.Line, Column: d.Loc.Start.Column},
					End:      Location{Line: d.Loc.End.Line, Column: d.Loc.End.Column},
				},
			}
		}
	}
	return r
}

type instance struct {
	mod *lightningcss_wasm.Module
	h   *host
}

func newInstance() *instance {
	h := new(host)
	mod := lightningcss_wasm.New(h)
	return &instance{mod: mod, h: h}
}

func (in *instance) mem() []byte {
	return *in.mod.Xmemory().Slice()
}

func (in *instance) alloc(n int) (int32, error) {
	if n == 0 {
		return 0, nil
	}
	ptr := in.mod.Xlcss_alloc(int32(n))
	if ptr == 0 {
		return 0, errors.New("lightningcss: out of wasm memory")
	}
	return ptr, nil
}

func (in *instance) writeBytes(b []byte) (int32, error) {
	ptr, err := in.alloc(len(b))
	if err != nil {
		return 0, err
	}
	copy(in.mem()[uint32(ptr):], b)
	return ptr, nil
}

func (in *instance) readBytes(ptr, n uint32) []byte {
	if n == 0 {
		return nil
	}
	out := make([]byte, n)
	copy(out, in.mem()[ptr:ptr+n])
	return out
}

// readResult decodes the eight-slot result quad written by lcss_transform.
func (in *instance) readResult(outPtr int32) (*Result, error) {
	mem := in.mem()
	u := func(i uint32) uint32 { return binary.LittleEndian.Uint32(mem[uint32(outPtr)+i*4:]) }
	codePtr, codeLen := u(0), u(1)
	mapPtr, mapLen := u(2), u(3)
	metaPtr, metaLen := u(4), u(5)
	errPtr, errLen := u(6), u(7)

	if errLen != 0 {
		return nil, errors.New(string(in.readBytes(errPtr, errLen)))
	}

	var res *Result
	if metaLen != 0 {
		var m lightningcss_wasm.Meta
		if err := json.Unmarshal(in.readBytes(metaPtr, metaLen), &m); err != nil {
			return nil, err
		}
		res = result(&m)
	} else {
		res = &Result{}
	}
	res.Code = in.readBytes(codePtr, codeLen)
	res.Map = in.readBytes(mapPtr, mapLen)
	return res, nil
}

// mapValues returns a new map with f applied to each value of m, or nil if m is
// empty.
func mapValues[K comparable, V, W any](m map[K]V, f func(V) W) map[K]W {
	if len(m) == 0 {
		return nil
	}
	r := make(map[K]W, len(m))
	for k, v := range m {
		r[k] = f(v)
	}
	return r
}

func cssModuleExport(e lightningcss_wasm.CSSModuleExport) CSSModuleExport {
	x := CSSModuleExport{Name: e.Name, IsReferenced: e.IsReferenced}
	if len(e.Composes) != 0 {
		x.Composes = make([]CSSModuleReference, len(e.Composes))
		for i, c := range e.Composes {
			x.Composes[i] = CSSModuleReference(c)
		}
	}
	return x
}

func (in *instance) call(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			msg := in.h.stderr.String()
			if msg == "" {
				switch v := r.(type) {
				case error:
					msg = v.Error()
				case string:
					msg = v
				default:
					msg = "wasm panic"
				}
			}
			err = errors.New("lightningcss: " + msg)
		}
	}()
	return fn()
}

// host is a minimal implementation of a subset of wasi_snapshot_preview1.
type host struct {
	mod    *lightningcss_wasm.Module
	stderr strings.Builder
}

func (h *host) Init(mod any) {
	h.mod = mod.(*lightningcss_wasm.Module)
}

func (h *host) mem() []byte {
	return *h.mod.Xmemory().Slice()
}

func (h *host) Xrandom_get(ptr, n int32) int32 {
	mem := h.mem()
	if _, err := rand.Read(mem[uint32(ptr) : uint32(ptr)+uint32(n)]); err != nil {
		return 29 // __WASI_ERRNO_IO
	}
	return 0
}

func (h *host) Xfd_write(fd, iovsPtr, iovsLen, nwrittenPtr int32) int32 {
	mem := h.mem()
	var total uint32
	for i := range iovsLen {
		base := uint32(iovsPtr) + uint32(i)*8
		ptr := binary.LittleEndian.Uint32(mem[base:])
		n := binary.LittleEndian.Uint32(mem[base+4:])
		if fd == 1 || fd == 2 {
			// just stdout/stderr so we can get panic messages from rust
			h.stderr.Write(mem[ptr : ptr+n])
		}
		total += n
	}
	binary.LittleEndian.PutUint32(mem[uint32(nwrittenPtr):], total)
	return 0
}

func (h *host) Xenviron_sizes_get(countPtr, sizePtr int32) int32 {
	mem := h.mem()
	binary.LittleEndian.PutUint32(mem[uint32(countPtr):], 0)
	binary.LittleEndian.PutUint32(mem[uint32(sizePtr):], 0)
	return 0
}

func (h *host) Xenviron_get(environPtr, bufPtr int32) int32 {
	return 0
}

func (h *host) Xproc_exit(code int32) {
	panic(trap{fmt.Sprintf("wasm called proc_exit(%d)", code)})
}

type trap struct{ msg string }

func (t trap) Error() string { return t.msg }
