FROM golang:1.22-alpine AS builder

ENV GOTOOLCHAIN=auto

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o backend-api ./cmd/server

FROM alpine:3.20

RUN apk --no-cache add ca-certificates

WORKDIR /srv
COPY --from=builder /app/backend-api ./backend-api

EXPOSE 8080

CMD ["./backend-api"]
