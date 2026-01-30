# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    make \
    gcc \
    musl-dev \
    mupdf-dev \
    tesseract-ocr-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 go build -o pdf-ocr-tool main.go

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    mupdf \
    tesseract-ocr \
    tesseract-ocr-data-eng \
    ca-certificates

# Copy binary from builder
COPY --from=builder /app/pdf-ocr-tool /usr/local/bin/pdf-ocr-tool

# Create working directory for PDFs
WORKDIR /pdfs

# Set entrypoint
ENTRYPOINT ["pdf-ocr-tool"]

# Default command (show help)
CMD ["--help"]