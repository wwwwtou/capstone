FROM golang:1.20-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o user .

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/user ./user
EXPOSE 8081
CMD ["/app/user"]
