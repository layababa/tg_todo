SHELL := /bin/bash

run-api:
	go run ./cmd/api

run-bot:
	go run ./cmd/bot

test:
	go test ./...

fmt:
	go fmt ./...

lint: fmt
	@echo "golangci-lint 未在本地配置，可在联网后安装运行。"

docker-build:
	docker build -t tg-miniapp-api -f Dockerfile .
