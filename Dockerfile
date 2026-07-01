FROM golang:1.25 AS builder
WORKDIR /app
COPY ./.git/ ./.git/
RUN git reset --hard HEAD
RUN go generate ./... && \
    CGO_ENABLED=0 go build \
        -ldflags "-s -w -X github.com/WangYihang/tranco-go-package/pkg/version.CommitHash=$(git rev-parse HEAD) -X github.com/WangYihang/tranco-go-package/pkg/version.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        -o /app/dist/tranco \
        ./cmd/tranco

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/dist/tranco /usr/local/bin/tranco
ENTRYPOINT [ "/usr/local/bin/tranco" ]