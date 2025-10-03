# Build Stage
FROM golang:1.23-alpine AS builder
WORKDIR /app

RUN apk add --no-cache git gcc musl-dev lame
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -a -o steganography-backend main.go

# Runtime
FROM alpine:latest
RUN apk --no-cache add ca-certificates lame
WORKDIR /root/

COPY --from=builder /app/steganography-backend .
RUN mkdir -p output

EXPOSE 8080
ENV PORT=8080
ENV OUTPUT_DIR=./output
ENV GIN_MODE=release

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/v1/health || exit 1

CMD ["./steganography-backend"]


