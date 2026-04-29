FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o url-shortener .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/url-shortener .
COPY --from=builder /app/frontend ./frontend

EXPOSE 8081

CMD ["./url-shortener"]
