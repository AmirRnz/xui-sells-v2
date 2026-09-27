# Stage 1: Build React Web Panel
FROM node:22-alpine AS web-builder
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Stage 2: Build Go Modular Monolith Binary
FROM golang:1.24-alpine AS go-builder
WORKDIR /app
RUN apk add --no-cache git ca-certificates tzdata
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copy built web assets to web/dist for embedding
COPY --from=web-builder /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/bin/xui-sells ./cmd/xui-sells

# Stage 3: Minimal Production Image
FROM alpine:3.21 AS runner
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata tar gzip
COPY --from=go-builder /app/bin/xui-sells /usr/local/bin/xui-sells
COPY migrations /app/migrations

# Default environment configuration
ENV PORT=8080 \
    APP_ENV=production

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/xui-sells"]
CMD ["run"]
