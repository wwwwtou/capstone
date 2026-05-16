# Build stage
FROM golang:1.21-alpine AS builder
WORKDIR /src

# Install git (for go modules) and ca-certificates
RUN apk add --no-cache git ca-certificates

COPY go.mod ./
# Ensure required modules are resolved (some are marked indirect in go.mod)
RUN go get github.com/gin-gonic/gin@v1.9.1
RUN go mod download
RUN go mod tidy

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /usr/local/bin/rec-service ./cmd/api

# Final image
FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /usr/local/bin/rec-service /usr/local/bin/rec-service
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/rec-service"]
