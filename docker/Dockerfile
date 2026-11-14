FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/teralur ./cmd/main.go

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 app \
    && mkdir -p /app/config/logs/query_log \
    && chown -R app:app /app
WORKDIR /app
COPY --from=build /out/teralur /app/teralur
COPY database/seeders/json /app/database/seeders/json
USER app
EXPOSE 8888
# Migrasi dan seeder idempoten, jadi aman dijalankan setiap container start.
CMD ["/app/teralur", "--migrate:run", "--seed", "--run"]
