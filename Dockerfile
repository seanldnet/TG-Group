FROM --platform=$BUILDPLATFORM node:22.18-bookworm-slim AS web-build
WORKDIR /src
RUN corepack enable && corepack prepare pnpm@11.16.0 --activate
COPY . .
RUN pnpm install --frozen-lockfile
RUN pnpm --filter @vben/web-antd run build

FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS api-build
WORKDIR /src/backend
COPY backend/ ./
RUN go mod tidy
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -trimpath -ldflags="-s -w" -o /out/tg-group ./cmd/server

FROM alpine:3.21
RUN apk add --no-cache ca-certificates wget \
    && addgroup -S app \
    && adduser -S -G app app
WORKDIR /app
COPY --from=api-build /out/tg-group /usr/local/bin/tg-group
COPY --from=web-build /src/apps/web-antd/dist /app/web
ENV WEB_DIR=/app/web
USER app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/tg-group"]
