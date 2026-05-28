# Stage 1 — compile Go binary
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod .
COPY main.go .
RUN go build -o order-service .

# Stage 2 — minimal runtime
FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/order-service .
EXPOSE 3003
CMD ["./order-service"]
