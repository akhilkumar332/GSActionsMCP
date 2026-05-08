# Build Stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Ensure go.mod and go.sum are updated with manual additions
RUN go mod tidy

# Build the binary, stripping debug info to keep it small
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/mcp-server ./cmd/server

# Run Stage
# Using scratch for an ultra-small image, or alpine if we need CA certs
FROM alpine:latest

# Need CA certificates for remote API calls if the SDK makes any HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /

COPY --from=builder /go/bin/mcp-server /mcp-server

# Expose port (default 8080)
EXPOSE 8080

ENTRYPOINT ["/mcp-server"]
