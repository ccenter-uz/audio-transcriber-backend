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
# COPY --from=builder /app/internal/media /app/internal/media
# COPY --from=builder /app/internal/media/audio /app/internal/media/audio
# COPY --from=builder /app/internal/media/segments /app/internal/media/segments
COPY --from=builder /app/internal/controller/http/casbin/model.conf ./internal/controller/http/casbin/
COPY --from=builder /app/internal/controller/http/casbin/policy.csv ./internal/controller/http/casbin/

RUN chmod +x /app/voice_transcribe

EXPOSE 8080

CMD ["/app/voice_transcribe"]
