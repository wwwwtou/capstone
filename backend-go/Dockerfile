# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git ca-certificates

COPY . .
RUN go mod init backend-go || true
RUN go mod tidy || true
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /usr/local/bin/rec-service ./

# Final image
FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /usr/local/bin/rec-service /usr/local/bin/rec-service
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/rec-service"]
