# Multi Arch Dockerfile for Docker Buildx
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-s -w" -o osp3-exporter main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/osp3-exporter .

EXPOSE 9120

ENTRYPOINT ["./osp3-exporter"]