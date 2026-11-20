FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/tg-group ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates wget && addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=build /out/tg-group /usr/local/bin/tg-group
USER app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/tg-group"]
