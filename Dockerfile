FROM rust:1.94.0-alpine3.23 AS build

WORKDIR /build

RUN apk add --no-cache elfutils pkgconfig perl make just upx musl-dev

COPY . .

RUN just test-all && \
    cargo build --bin sm --release && \
    eu-elfcompress target/release/sm && \
    strip target/release/sm && \
    upx -9 --lzma target/release/sm && \
    chmod +x target/release/sm

FROM scratch

COPY --from=build /build/target/release/sm /sm

ENTRYPOINT ["/sm"]
