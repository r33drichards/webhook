FROM golang:1.24-bookworm AS webhook
WORKDIR /src
COPY . .
RUN go build -trimpath -o /webhook .

FROM rust:1.100-bookworm AS component
ARG WREV_REVISION=4fa2a762f07365efa5366747cf9122427b7013c0
RUN test -n "$WREV_REVISION" && \
    git init /wrev && git -C /wrev fetch --depth 1 https://github.com/r33drichards/wrev.git "$WREV_REVISION" && \
    git -C /wrev checkout --detach FETCH_HEAD
RUN rustup target add wasm32-wasip2 && \
    curl -fsSL --retry 3 https://github.com/WebAssembly/wasi-sdk/releases/download/wasi-sdk-24/wasi-sdk-24.0-x86_64-linux.tar.gz | tar -xz -C /opt
ENV WASI_SDK=/opt/wasi-sdk-24.0-x86_64-linux
ENV CC_wasm32_wasip2=/opt/wasi-sdk-24.0-x86_64-linux/bin/clang
WORKDIR /wrev
RUN cargo build --locked --release --target wasm32-wasip2 --features sqlite-backend --bin wrev

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl xz-utils \
    && curl -fsSL --retry 3 https://github.com/bytecodealliance/wasmtime/releases/download/v51.0.0/wasmtime-v51.0.0-x86_64-linux.tar.xz | tar -xJ -C /opt \
    && ln -s /opt/wasmtime-v51.0.0-x86_64-linux/wasmtime /usr/local/bin/wasmtime \
    && apt-get purge -y --auto-remove curl xz-utils && rm -rf /var/lib/apt/lists/*
COPY --from=webhook /webhook /usr/local/bin/webhook
COPY --from=component /wrev/target/wasm32-wasip2/release/wrev.wasm /opt/wrev.wasm
ENV XDG_CACHE_HOME=/tmp
ENTRYPOINT ["sh", "-c", "umask 077; exec webhook -port \"${PORT:-8080}\" -wrev-component /opt/wrev.wasm"]
