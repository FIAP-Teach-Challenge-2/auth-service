FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/auth-service .

FROM alpine:3.20 AS runtime

RUN apk add --no-cache ca-certificates wget \
    && addgroup -S app \
    && adduser -S app -G app

WORKDIR /app

COPY --from=builder /out/auth-service /app/auth-service

USER app

EXPOSE 8001

ENTRYPOINT ["/app/auth-service"]
