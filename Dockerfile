FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/focus \
    ./cmd/focus


FROM alpine:3.22

WORKDIR /app

COPY --from=builder /out/focus /app/focus

EXPOSE 8080

CMD ["/app/focus", "api"]