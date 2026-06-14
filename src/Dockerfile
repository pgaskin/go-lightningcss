FROM debian:trixie AS base

ARG WASI_SDK_VERSION=33
ARG BINARYEN_VERSION=129

ARG WASI_SDK_CHECKSUM=sha256:0ba8b5bfaeb2adf3f29bab5841d76cf5318ab8e1642ea195f88baba1abd47bce
ARG BINARYEN_CHECKSUM=sha256:50b9fa62b9abea752da92ec57e0c555fee578760cd237c40107957715d2976ba

ARG WASI_SDK_URL=https://github.com/WebAssembly/wasi-sdk/releases/download/wasi-sdk-${WASI_SDK_VERSION}/wasi-sdk-${WASI_SDK_VERSION}.0-x86_64-linux.tar.gz
ARG BINARYEN_URL=https://github.com/WebAssembly/binaryen/releases/download/version_${BINARYEN_VERSION}/binaryen-version_${BINARYEN_VERSION}-x86_64-linux.tar.gz

FROM base AS wasi-sdk
ADD --checksum=${WASI_SDK_CHECKSUM} ${WASI_SDK_URL} ./wasi-sdk.tar.gz
RUN mkdir /opt/wasi-sdk && tar -C /opt/wasi-sdk --strip-components=1 -xf /wasi-sdk.tar.gz

FROM base AS binaryen
ADD --checksum=${BINARYEN_CHECKSUM} ${BINARYEN_URL} ./binaryen.tar.gz
RUN mkdir /opt/binaryen && tar -C /opt/binaryen --strip-components=1 -xf binaryen.tar.gz

FROM golang:1.26-trixie AS wasm2go
RUN GOPROXY=direct GONOSUMDB=github.com/ncruces/wasm2go GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=url.https://github.com/pgaskin/wasm2go.insteadOf GIT_CONFIG_VALUE_0=https://github.com/ncruces/wasm2go \
    GOBIN=/ go install -trimpath -v github.com/ncruces/wasm2go@f58c5dc11b49225fd336f1ad2d56b5001a47fc51

FROM rust:1.96.0-trixie AS rust-base
RUN rustup target add wasm32-wasip1
COPY --link --from=wasi-sdk /opt /opt
COPY --link --from=binaryen /opt /opt
ENV PATH="/opt/wasi-sdk/bin:/opt/binaryen/bin:$PATH"

FROM rust-base AS build

WORKDIR /build
COPY --link Cargo.toml Cargo.lock /build/
COPY --link src /build/src

ENV RUSTFLAGS="-C target-feature=+bulk-memory,+mutable-globals,+sign-ext,+nontrapping-fptoint"
RUN cargo build --release --target wasm32-wasip1 --locked

RUN wasm-opt -g target/wasm32-wasip1/release/lightningcss_shim.wasm -o lightningcss.opt.wasm -Oz \
 --enable-bulk-memory --enable-mutable-globals --enable-sign-ext --enable-nontrapping-float-to-int \
 --strip-debug --strip-dwarf

COPY --link --from=wasm2go /wasm2go /usr/local/bin/
RUN mkdir -p internal
RUN wasm2go -embed -maxfunc=15000 -pkg lightningcss_wasm -o internal/lightningcss.go lightningcss.opt.wasm

FROM scratch
COPY --link --from=build /build/internal/ /internal/
