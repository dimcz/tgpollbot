FROM golang:1.18 AS builder
COPY . /src
WORKDIR /src
RUN VERSION=`date -u +%Y-%m-%dT%H:%M:%SZ`
RUN GOOS=linux GOARCH=amd64 go build -o /service -ldflags "-X main.version=$VERSION" ./main.go

FROM alpine:latest
COPY --from=builder /service /service

ENTRYPOINT ["/service"]
