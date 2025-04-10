FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags migrate -o voice_transcribe ./cmd

FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/voice_transcribe /app/voice_transcribe
COPY --from=builder /app/config /app/config
COPY --from=builder /app/migrations /app/migrations

RUN chmod +x /app/voice_transcribe

EXPOSE 8080

CMD ["/app/voice_transcribe"]
