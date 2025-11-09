# syntax=docker/dockerfile:1.6

FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod ./
# 预下载依赖（若无第三方依赖会快速完成）
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/bot ./cmd/bot

FROM gcr.io/distroless/base-debian12 AS runner
WORKDIR /app
ENV SERVER_PORT=8080
COPY --from=builder /bin/api /bin/api
COPY --from=builder /bin/bot /bin/bot
EXPOSE 8080
CMD ["/bin/api"]
