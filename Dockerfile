FROM golang:1.22 AS builder
RUN go install github.com/goreleaser/goreleaser@latest
WORKDIR /app
COPY ./.git/ ./.git/
RUN git reset --hard HEAD
RUN goreleaser build --clean --id=tranco --snapshot

FROM alpine:3.14
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/dist/tranco_linux_amd64_v1/tranco /usr/local/bin/tranco
ENTRYPOINT [ "/usr/local/bin/tranco" ]