FROM golang:1.26-alpine3.23 AS app-build

WORKDIR /build

RUN apk --no-cache add upx

COPY go.mod go.sum ./
RUN go mod download

COPY . /build

ARG VERSION=dev

RUN CGO_ENABLED=0 go build -ldflags="-w -s -X main.version=${VERSION}" -o sm ./cmd/sm && \
    upx -9 --lzma sm && \
    chmod +x sm

FROM scratch

WORKDIR /app

COPY --from=app-build --chown=10001:10001 /build/sm /app/sm

USER 10001:10001

ENTRYPOINT ["/app/sm"]
