FROM golang:1.18-alpine AS builder

WORKDIR /app

COPY go.mod .
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -a -o /tgpollbot -ldflags "-X main.VERSION=`date -u +%Y-%m-%dT%H:%M:%SZ`"

FROM alpine
WORKDIR /
COPY --from=builder /tgpollbot /tgpollbot

EXPOSE 8080

ENTRYPOINT ["/tgpollbot"]
