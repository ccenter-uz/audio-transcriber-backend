FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags migrate -o voice_transcribe ./cmd

FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/voice_transcribe /app/voice_transcribe
COPY --from=builder /app/config /app/config
COPY --from=builder /app/migrations /app/migrations
COPY --from=builder /app/internal/controller/http/casbin/model.conf ./internal/controller/http/casbin/
COPY --from=builder /app/internal/controller/http/casbin/policy.csv ./internal/controller/http/casbin/

ENV TZ=Asia/Tashkent
RUN ln -snf /usr/share/zoneinfo/Asia/Tashkent /etc/localtime && echo "Asia/Tashkent" > /etc/timezone

RUN chmod +x /app/voice_transcribe

EXPOSE 8081

CMD ["/app/voice_transcribe"]
