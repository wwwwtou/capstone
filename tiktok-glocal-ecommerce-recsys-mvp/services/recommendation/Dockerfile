FROM golang:1.20-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o recommendation .

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/recommendation ./recommendation
EXPOSE 8083
CMD ["/app/recommendation"]
