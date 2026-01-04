FROM rust:1.92.0-alpine3.23 AS build

WORKDIR /build

RUN apk add elfutils pkgconfig perl make just upx

COPY . .

RUN just test-all && \
    RUSTFLAGS='-C target-feature=+crt-static' cargo build --bin sm --release && \
    eu-elfcompress target/release/sm && \
    strip target/release/sm && \
    upx -9 --lzma target/release/sm && \
    chmod +x target/release/sm

FROM scratch

COPY --from=build /build/target/release/sm /sm
