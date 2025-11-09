# syntax=docker/dockerfile:1
FROM golang:1.25 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.version=${VERSION:-dev}" -o /out/conn-spotify ./cmd/conn-spotify

FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=builder /out/conn-spotify /conn-spotify
EXPOSE 8081
USER 65532:65532
ENTRYPOINT ["/conn-spotify"]
