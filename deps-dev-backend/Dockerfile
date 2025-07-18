# Stage 1: Build with full Go + GCC support
FROM golang:1.24-bookworm as builder

WORKDIR /app

# Enable CGO explicitly
ENV CGO_ENABLED=1
ENV GOOS=linux

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o server .

# Stage 2: Minimal runtime
FROM debian:bookworm-slim

WORKDIR /app

# Required for go-sqlite3 runtime
RUN apt-get update && apt-get install -y --no-install-recommends \
    libsqlite3-0 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/server .

EXPOSE 8080

CMD ["./server"]
