FROM golang:1.20-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o gateway .

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/gateway ./gateway
EXPOSE 8080
CMD ["/app/gateway"]
