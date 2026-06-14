package lightningcss_wasm

type Request struct {
	Filename               string            `json:"filename,omitempty"`
	Minify                 bool              `json:"minify,omitempty"`
	SourceMap              bool              `json:"sourceMap,omitempty"`
	InputSourceMap         string            `json:"inputSourceMap,omitempty"`
	ProjectRoot            string            `json:"projectRoot,omitempty"`
	Targets                *TargetsReq       `json:"targets,omitempty"`
	ErrorRecovery          bool              `json:"errorRecovery,omitempty"`
	Nesting                bool              `json:"nesting,omitempty"`
	CustomMedia            bool              `json:"customMedia,omitempty"`
	DeepSelectorCombinator bool              `json:"deepSelectorCombinator,omitempty"`
	UnusedSymbols          []string          `json:"unusedSymbols,omitempty"`
	CSSModules             *CSSModulesReq    `json:"cssModules,omitempty"`
	AnalyzeDependencies    bool              `json:"analyzeDependencies,omitempty"`
	RemoveImports          bool              `json:"removeImports,omitempty"`
	PseudoClasses          *PseudoClassesReq `json:"pseudoClasses,omitempty"`
	StyleAttribute         bool              `json:"styleAttribute,omitempty"`
}

type TargetsReq struct {
	Android   uint32 `json:"android,omitempty"`
	Chrome    uint32 `json:"chrome,omitempty"`
	Edge      uint32 `json:"edge,omitempty"`
	Firefox   uint32 `json:"firefox,omitempty"`
	IE        uint32 `json:"ie,omitempty"`
	IOSSafari uint32 `json:"iosSaf,omitempty"`
	Opera     uint32 `json:"opera,omitempty"`
	Safari    uint32 `json:"safari,omitempty"`
	Samsung   uint32 `json:"samsung,omitempty"`
}

type CSSModulesReq struct {
	Pattern      string `json:"pattern,omitempty"`
	DashedIdents bool   `json:"dashedIdents,omitempty"`
	Pure         bool   `json:"pure,omitempty"`
	Animation    *bool  `json:"animation,omitempty"`
	Grid         *bool  `json:"grid,omitempty"`
	CustomIdents *bool  `json:"customIdents,omitempty"`
	Container    *bool  `json:"container,omitempty"`
}

type PseudoClassesReq struct {
	Hover        string `json:"hover,omitempty"`
	Active       string `json:"active,omitempty"`
	Focus        string `json:"focus,omitempty"`
	FocusVisible string `json:"focusVisible,omitempty"`
	FocusWithin  string `json:"focusWithin,omitempty"`
}

type Meta struct {
	Exports      CSSModuleExports    `json:"exports"`
	References   CSSModuleReferences `json:"references"`
	Dependencies []Dependency        `json:"dependencies"`
	Warnings     []string            `json:"warnings"`
}

type Dependency struct {
	Type        string      `json:"type"`
	URL         string      `json:"url"`
	Placeholder string      `json:"placeholder"`
	Supports    string      `json:"supports,omitempty"`
	Media       string      `json:"media,omitempty"`
	Loc         SourceRange `json:"loc"`
}

type SourceRange struct {
	FilePath string      `json:"filePath"`
	Start    LocationRes `json:"start"`
	End      LocationRes `json:"end"`
}

type LocationRes struct {
	Line   uint32 `json:"line"`
	Column uint32 `json:"column"`
}

type CSSModuleExports map[string]CSSModuleExport

type CSSModuleExport struct {
	Name         string               `json:"name"`
	Composes     []CSSModuleReference `json:"composes"`
	IsReferenced bool                 `json:"isReferenced"`
}

type CSSModuleReferences map[string]CSSModuleReference

type CSSModuleReference struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	Specifier string `json:"specifier,omitempty"`
}
