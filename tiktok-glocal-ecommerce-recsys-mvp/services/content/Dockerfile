FROM golang:1.20-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o content .

FROM alpine:3.18
WORKDIR /app
COPY --from=builder /app/content ./content
EXPOSE 8082
CMD ["/app/content"]
