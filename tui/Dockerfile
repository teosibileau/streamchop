FROM golang:1.23-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /streamchop-setup .

FROM alpine:3.21
WORKDIR /app
COPY --from=builder /streamchop-setup /usr/local/bin/streamchop-setup
ENTRYPOINT ["streamchop-setup"]
