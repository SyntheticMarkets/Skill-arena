FROM golang:1.25.0-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /skill-arena ./cmd/api

FROM scratch
COPY --from=builder /skill-arena /skill-arena
EXPOSE 8080
ENTRYPOINT ["/skill-arena"]
