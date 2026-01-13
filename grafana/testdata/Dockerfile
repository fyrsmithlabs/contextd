FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o metrics-generator ./grafana/testdata/generate_metrics.go

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/metrics-generator .
EXPOSE 9090
CMD ["./metrics-generator"]
