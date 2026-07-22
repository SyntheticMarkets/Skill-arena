FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /skill-arena ./cmd/api

FROM alpine:3.22
RUN apk add --no-cache ca-certificates wget && addgroup -S skillarena && adduser -S -G skillarena skillarena
COPY --from=builder /skill-arena /usr/local/bin/skill-arena
USER skillarena
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/skill-arena"]
