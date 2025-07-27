FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY . .

RUN go build -o rinha

FROM alpine:3.19.0

COPY --from=builder /app/rinha /app/rinha