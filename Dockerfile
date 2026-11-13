# --- UI ---
FROM node:22-alpine AS web
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# --- binary ---
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /app/web/dist ./internal/webui/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/crew ./cmd/crew

# --- runtime ---
FROM alpine:3.20
RUN adduser -D -H -u 10001 crew \
    && mkdir -p /data \
    && chown crew:crew /data \
    && apk add --no-cache ca-certificates
COPY --from=build /out/crew /usr/local/bin/crew
USER crew
VOLUME ["/data"]
ENV CREW_ADDR=0.0.0.0:8787 \
    CREW_DB=/data/crew.db \
    CREW_ENV=production
EXPOSE 8787
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s \
  CMD wget -qO- http://127.0.0.1:8787/api/health >/dev/null || exit 1
ENTRYPOINT ["/usr/local/bin/crew"]
