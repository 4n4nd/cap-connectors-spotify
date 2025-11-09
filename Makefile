APP=conn-spotify
VERSION?=dev

.PHONY: run build test tidy docker

run:
	go run ./cmd/conn-spotify

build:
	CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=$(VERSION)" -o ./bin/$(APP) ./cmd/conn-spotify

test:
	go test ./...

tidy:
	go mod tidy

docker:
	docker build -t ghcr.io/4n4nd/cap-connectors-spotify:$(VERSION) .
