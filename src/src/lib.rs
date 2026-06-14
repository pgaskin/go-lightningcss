use std::alloc::{alloc as rust_alloc, dealloc as rust_dealloc, Layout};
use std::sync::{Arc, RwLock};

use lightningcss::stylesheet::{ParserOptions, StyleAttribute, StyleSheet};
use parcel_sourcemap::SourceMap;

mod types;
use types::*;

#[no_mangle]
pub extern "C" fn lcss_transform(
    src_ptr: *const u8,
    src_len: u32,
    req_ptr: *const u8,
    req_len: u32,
    out: *mut u32,
) -> i32 {
    let src = unsafe {
        std::str::from_utf8_unchecked(std::slice::from_raw_parts(src_ptr, src_len as usize))
    };
    let req_json = unsafe {
        std::str::from_utf8_unchecked(std::slice::from_raw_parts(req_ptr, req_len as usize))
    };
    let result = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| run(src, req_json)));
    let (vals, rc): ([u32; 8], i32) = match result {
        Ok(Ok(o)) => {
            let code = leak(o.code);
            let map = leak(o.map);
            let meta = leak(o.meta);
            ([code.0, code.1, map.0, map.1, meta.0, meta.1, 0, 0], 0)
        }
        Ok(Err(msg)) => {
            let err = leak(msg.into_bytes());
            ([0, 0, 0, 0, 0, 0, err.0, err.1], 1)
        }
        Err(_) => {
            let err = leak(b"lightningcss: internal panic".to_vec());
            ([0, 0, 0, 0, 0, 0, err.0, err.1], 1)
        }
    };
    unsafe {
        for (i, v) in vals.iter().enumerate() {
            *out.add(i) = *v;
        }
    }
    rc
}

fn run(src: &str, req_json: &str) -> Result<Output, String> {
    let req: Request =
        serde_json::from_str(req_json).map_err(|e| format!("invalid options: {e}"))?;

    let pseudo_classes_req = req.pseudo_classes.clone().unwrap_or_default(); // owned strings must outlive to_css
    let pseudo_classes = pseudo_classes_req.into_pseudo_classes();
    let targets = req.targets();

    if req.style_attribute {
        let mut attr =
            StyleAttribute::parse(src, ParserOptions::default()).map_err(|e| e.to_string())?;
        attr.minify(req.minify_options(targets));
        let res = attr
            .to_css(req.printer_options(targets, None, pseudo_classes))
            .map_err(|e| e.to_string())?;
        let meta = Meta {
            exports: &None,
            references: &None,
            dependencies: &res.dependencies,
            warnings: Vec::new(),
        };
        let meta = serde_json::to_vec(&meta).map_err(|e| e.to_string())?;
        return Ok(Output {
            code: res.code.into_bytes(),
            map: Vec::new(),
            meta,
        });
    }

    let warnings = Arc::new(RwLock::new(Vec::new()));
    let parser_opts = req.parser_options(warnings.clone())?;

    let mut sheet = StyleSheet::parse(src, parser_opts).map_err(|e| e.to_string())?;

    sheet
        .minify(req.minify_options(targets))
        .map_err(|e| e.to_string())?;

    let mut source_map = if req.source_map {
        let mut sm = SourceMap::new("/");
        sm.add_source(&sheet.sources[0]);
        sm.set_source_content(0, src).map_err(|e| e.to_string())?;
        Some(sm)
    } else {
        None
    };

    let printer_opts = req.printer_options(targets, source_map.as_mut(), pseudo_classes);

    let res = sheet.to_css(printer_opts).map_err(|e| e.to_string())?;

    let map = match source_map {
        Some(mut sm) => {
            if let Some(input) = &req.input_source_map {
                let mut from = SourceMap::from_json("/", input).map_err(|e| e.to_string())?;
                sm.extends(&mut from).map_err(|e| e.to_string())?;
            }
            sm.to_json(None).map_err(|e| e.to_string())?.into_bytes()
        }
        None => Vec::new(),
    };

    let meta = Meta {
        exports: &res.exports,
        references: &res.references,
        dependencies: &res.dependencies,
        warnings: warnings
            .read()
            .unwrap()
            .iter()
            .map(|w| w.to_string())
            .collect(),
    };
    let meta = serde_json::to_vec(&meta).map_err(|e| e.to_string())?;

    Ok(Output {
        code: res.code.into_bytes(),
        map,
        meta,
    })
}

#[no_mangle]
pub extern "C" fn lcss_alloc(len: u32) -> *mut u8 {
    if len == 0 {
        return std::ptr::null_mut();
    }
    unsafe { rust_alloc(Layout::from_size_align_unchecked(len as usize, 1)) }
}

#[no_mangle]
pub extern "C" fn lcss_free(ptr: *mut u8, len: u32) {
    // len MUST be exact
    if ptr.is_null() || len == 0 {
        return;
    }
    unsafe { rust_dealloc(ptr, Layout::from_size_align_unchecked(len as usize, 1)) }
}

fn leak(bytes: Vec<u8>) -> (u32, u32) {
    if bytes.is_empty() {
        return (0, 0);
    }
    let boxed = bytes.into_boxed_slice();
    let len = boxed.len() as u32;
    let ptr = Box::into_raw(boxed) as *mut u8 as u32;
    (ptr, len)
}
