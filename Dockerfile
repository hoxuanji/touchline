# syntax=docker/dockerfile:1

# 1) Build the SPA.
FROM node:22-alpine AS web
WORKDIR /app/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

# 2) Build the static Go binary with the freshly built SPA embedded.
FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Must come after COPY . . so the freshly-built SPA wins over any dist remnants in the context.
COPY --from=web /app/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /touchline ./cmd/touchline

# 3) Ship the binary with CA certs (needed for TLS to api-sports.io).
FROM gcr.io/distroless/static:nonroot
COPY --from=build /touchline /touchline
ENV PORT=8080
EXPOSE 8080
ENTRYPOINT ["/touchline"]
