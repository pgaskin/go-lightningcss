# go-lightningcss

Go bindings for [LightningCSS](https://github.com/parcel-bundler/lightningcss).

This library wraps a WebAssembly build of LightningCSS transpiled to Go using [wasm2go](https://github.com/ncruces/wasm2go).

> [!WARNING]
>
> These bindings are still experimental and are subject to change. They also produce extremely large binaries and are slow to compile.
>
> An experimental (and mostly untested) version of wasm2go was used to attempt to make it compile more efficiently.

### Getting started

```go
res, err := lightningcss.Transform(`...`, &lightningcss.Options{
    Filename: "style.css",
    Minify:   true,
    Nesting:  true,
    Targets: lightningcss.Targets{
        Chrome: lightningcss.Version(95, 0, 0),
        Safari: lightningcss.Version(14, 0, 0),
    },
    SourceMap: true,
})
if err != nil {
    panic(err)
}
for _, w := range res.Warnings {
    fmt.Fprintln(os.Stderr, "warning:", w)
}
fmt.Printf("%s\n", res.Code)
```

### Design

Most LightningCSS functionality is exposed. All errors are handled appropriately and returned.

### Concurrency

The package is safe for concurrent use since each call instantiates its own temporary copy of the module.

### Testing

The generated code should be [reproducible](./src/Dockerfile), but I haven't fully verified it (I'm yet familiar enough with Rust to fully verify it).

The CSS output will be identical to other LightningCSS bindings.

### Building

The transpiled code is very large, and compiling it takes a LOT of memory. Currently, it takes ~12 GB of RAM to build on an 8-core machine with Go 1.26.2. This appears to be mostly because the cost for Go's optimizer is super-linear to the function size and runs concurrently. The memory usage also increases with the total size of the package, which is quite large.

You can set `GOMAXPROCS` to a lower value when building to reduce the memory usage significantly, though it will stail take multiple gigabytes.

You can also set `-gcflags=all='-N -l'` to disable optimizations, which reduces the memory a bit more at the cost of slower, bloated (multi-gigabyte) binaries.

If you're building on a smaller machine, I recommend using something like `systemd-run --user --scope -p MemoryMax=12G -p MemorySwapMax=0` to ensure you don't accidentally OOM your session.

As long as the wasm2go blob isn't modified, future builds should be nearly instant due to caching.

This overhead is only for compiling. Although the resulting binary is about 10x the size of the WebAssembly, the memory usage at runtime is reasonable.
