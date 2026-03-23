FROM rust:1.84-alpine AS builder

RUN apk add --no-cache musl-dev

WORKDIR /build
COPY Cargo.toml Cargo.lock* ./
COPY src ./src

RUN cargo build --release

FROM alpine:3.21

COPY --from=builder /build/target/release/streamchop-emitter /usr/local/bin/streamchop-emitter

ENTRYPOINT ["streamchop-emitter"]
