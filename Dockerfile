FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o payment-processor
FROM alpine:3.19.0
COPY --from=builder /app/payment-processor /app/payment-processor
WORKDIR /app
CMD ["./payment-processor"]
