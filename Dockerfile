FROM golang:alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /api ./cmd/api
RUN go build -o /consumer ./cmd/consumer
RUN go build -o /producer ./cmd/producer

FROM alpine:latest
WORKDIR /
COPY --from=builder /api /api
COPY --from=builder /consumer /consumer
COPY --from=builder /producer /producer

EXPOSE 8080
CMD ["/api"]
