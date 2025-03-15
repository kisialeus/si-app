FROM golang:1.21 AS builder

WORKDIR /app
COPY main.go .

RUN go mod init myapp && go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server .

FROM alpine:latest AS worklayer

WORKDIR /app
COPY --from=builder /app/server .

RUN chmod +x /app/server

EXPOSE 80
CMD ["/app/server"]
