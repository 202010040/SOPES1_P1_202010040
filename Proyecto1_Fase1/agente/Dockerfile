FROM golang:1.19-alpine AS builder

WORKDIR /app
COPY . .

RUN go mod init agente-monitoreo
RUN go mod tidy
RUN go build -o agente

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/agente .
EXPOSE 8080
CMD ["./agente"]
