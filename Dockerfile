FROM golang:1.21
WORKDIR /app/
COPY ./go.mod /app/
COPY ./go.sum /app/
COPY ./cmd/ /app/cmd/
COPY ./pkg/ /app/pkg/
COPY ./tool/ /app/tool/
RUN go env -w GO111MODULE=on
RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN go mod download
RUN go build -o tranco-server /app/cmd/server/main.go
ENTRYPOINT [ "/app/tranco-server" ]
