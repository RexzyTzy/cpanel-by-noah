# Multi-stage build untuk Railway
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go.mod
COPY go.mod ./
RUN go mod download

# Copy source code
COPY index.go ./

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app index.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy binary dari builder
COPY --from=builder /app/app .

# Expose port
EXPOSE 8080

# Jalankan aplikasi
CMD ["./app"]
