FROM golang:1.23.0-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/taskservice ./cmd/api

FROM scratch

WORKDIR /app


COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /out/taskservice /app/taskservice
COPY --from=builder /src/web/static /app/web/static

EXPOSE 8080

ENTRYPOINT ["/app/taskservice"]