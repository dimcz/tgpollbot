FROM golang:1.18-alpine AS builder

WORKDIR /app

COPY go.mod .
RUN go mod download

COPY . .

RUN VERSION=`date -u +%Y-%m-%dT%H:%M:%SZ`
RUN CGO_ENABLED=0 go build -a -o /tgpollbot -ldflags "-X main.version=$VERSION"

FROM alpine
WORKDIR /
COPY --from=builder /tgpollbot /tgpollbot

EXPOSE 8080

ENTRYPOINT ["/tgpollbot"]
