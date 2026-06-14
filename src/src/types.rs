use std::sync::{Arc, RwLock};

use lightningcss::css_modules::{Config, CssModuleExports, CssModuleReferences, Pattern};
use lightningcss::dependencies::{Dependency, DependencyOptions};
use lightningcss::error::{Error, ParserError};
use lightningcss::printer::{PrinterOptions, PseudoClasses};
use lightningcss::stylesheet::{MinifyOptions, ParserFlags, ParserOptions};
use lightningcss::targets::{Browsers, Targets};
use parcel_sourcemap::SourceMap;
use serde::{Deserialize, Serialize};

#[derive(Deserialize, Default, Clone)]
#[serde(rename_all = "camelCase", default)]
pub struct Request {
    pub filename: String,
    pub minify: bool,
    pub source_map: bool,
    pub input_source_map: Option<String>,
    pub project_root: Option<String>,
    pub targets: Option<TargetsReq>,
    pub error_recovery: bool,
    pub nesting: bool,
    pub custom_media: bool,
    pub deep_selector_combinator: bool,
    pub unused_symbols: Vec<String>,
    pub css_modules: Option<CssModulesReq>,
    pub analyze_dependencies: bool,
    pub remove_imports: bool,
    pub pseudo_classes: Option<PseudoClassesReq>,
    pub style_attribute: bool,
}

#[derive(Deserialize, Default, Clone)]
#[serde(rename_all = "camelCase", default)]
pub struct TargetsReq {
    pub android: u32,
    pub chrome: u32,
    pub edge: u32,
    pub firefox: u32,
    pub ie: u32,
    pub ios_saf: u32,
    pub opera: u32,
    pub safari: u32,
    pub samsung: u32,
}

#[derive(Deserialize, Default, Clone)]
#[serde(rename_all = "camelCase", default)]
pub struct CssModulesReq {
    pub pattern: Option<String>,
    pub dashed_idents: bool,
    pub pure: bool,
    pub animation: Option<bool>,
    pub grid: Option<bool>,
    pub custom_idents: Option<bool>,
    pub container: Option<bool>,
}

#[derive(Deserialize, Default, PartialEq, Clone)]
#[serde(rename_all = "camelCase", default)]
pub struct PseudoClassesReq {
    pub hover: Option<String>,
    pub active: Option<String>,
    pub focus: Option<String>,
    pub focus_visible: Option<String>,
    pub focus_within: Option<String>,
}

impl Request {
    pub fn parser_flags(&self) -> ParserFlags {
        let mut flags = ParserFlags::empty();
        flags.set(ParserFlags::NESTING, self.nesting);
        flags.set(ParserFlags::CUSTOM_MEDIA, self.custom_media);
        flags.set(
            ParserFlags::DEEP_SELECTOR_COMBINATOR,
            self.deep_selector_combinator,
        );
        flags
    }
    pub fn dependency_options(&self) -> Option<DependencyOptions> {
        if self.analyze_dependencies {
            Some(DependencyOptions {
                remove_imports: self.remove_imports,
            })
        } else {
            None
        }
    }
    pub fn targets(&self) -> Targets {
        Targets {
            browsers: self.targets.clone().unwrap_or_default().into_browsers(),
            ..Default::default()
        }
    }
    pub fn minify_options(&self, targets: Targets) -> MinifyOptions {
        MinifyOptions {
            targets,
            unused_symbols: self.unused_symbols.iter().cloned().collect(),
        }
    }
    pub fn parser_options<'a, 'i>(
        &'a self,
        warnings: Arc<RwLock<Vec<Error<ParserError<'i>>>>>,
    ) -> Result<ParserOptions<'a, 'i>, String> {
        let css_modules = match &self.css_modules {
            Some(cm) => Some(cm.into_css_modules()?),
            None => None,
        };
        Ok(ParserOptions {
            filename: self.filename.clone(),
            css_modules,
            source_index: 0,
            error_recovery: self.error_recovery,
            warnings: Some(warnings),
            flags: self.parser_flags(),
        })
    }
    pub fn printer_options<'a>(
        &'a self,
        targets: Targets,
        source_map: Option<&'a mut SourceMap>,
        pseudo_classes: Option<PseudoClasses<'a>>,
    ) -> PrinterOptions<'a> {
        PrinterOptions {
            minify: self.minify,
            source_map,
            project_root: self.project_root.as_deref(),
            targets,
            analyze_dependencies: self.dependency_options(),
            pseudo_classes,
        }
    }
}

impl TargetsReq {
    pub fn into_browsers(self) -> Option<Browsers> {
        let b = Browsers {
            android: nz(self.android),
            chrome: nz(self.chrome),
            edge: nz(self.edge),
            firefox: nz(self.firefox),
            ie: nz(self.ie),
            ios_saf: nz(self.ios_saf),
            opera: nz(self.opera),
            safari: nz(self.safari),
            samsung: nz(self.samsung),
        };
        if b == Browsers::default() {
            None
        } else {
            Some(b)
        }
    }
}

impl CssModulesReq {
    pub fn into_css_modules(&self) -> Result<Config<'_>, String> {
        let mut config = Config::default();
        if let Some(p) = &self.pattern {
            config.pattern = Pattern::parse(p).map_err(|e| e.to_string())?;
        }
        config.dashed_idents = self.dashed_idents;
        config.pure = self.pure;
        if let Some(v) = self.animation {
            config.animation = v;
        }
        if let Some(v) = self.grid {
            config.grid = v;
        }
        if let Some(v) = self.custom_idents {
            config.custom_idents = v;
        }
        if let Some(v) = self.container {
            config.container = v;
        }
        Ok(config)
    }
}

impl PseudoClassesReq {
    pub fn into_pseudo_classes(&self) -> Option<PseudoClasses<'_>> {
        if *self == Self::default() {
            return None;
        }
        Some(PseudoClasses {
            hover: self.hover.as_deref(),
            active: self.active.as_deref(),
            focus: self.focus.as_deref(),
            focus_visible: self.focus_visible.as_deref(),
            focus_within: self.focus_within.as_deref(),
        })
    }
}

#[derive(Serialize)]
pub struct Meta<'a> {
    pub exports: &'a Option<CssModuleExports>,
    pub references: &'a Option<CssModuleReferences>,
    pub dependencies: &'a Option<Vec<Dependency>>,
    pub warnings: Vec<String>,
}

pub struct Output {
    pub code: Vec<u8>,
    pub map: Vec<u8>,
    pub meta: Vec<u8>,
}

fn nz(v: u32) -> Option<u32> {
    if v > 0 {
        Some(v)
    } else {
        None
    }
}
