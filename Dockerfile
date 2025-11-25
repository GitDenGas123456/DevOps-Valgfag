# syntax=docker/dockerfile:1

############################
# Builder
############################
FROM golang:1.25 AS build
WORKDIR /app

# Hent Go-moduler først (cache)
COPY go.mod go.sum ./
RUN go mod download

# Kopiér app-kilde
COPY . .

# Byg statisk binær (CGO fri -> nem container)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -p 1 -o app ./cmd/server


############################
# Runtime (lille image)
############################
FROM alpine:latest
WORKDIR /app

# Binær + assets
COPY --from=build /app/app /app/app
COPY --from=build /app/templates /app/templates
COPY --from=build /app/static /app/static

# Miljøvariabler (main.go læser disse)
ENV PORT=8080
ENV DATABASE_PATH=/app/data/seed/whoknows.db

# Appen opretter selv /app/data/seed ved runtime
EXPOSE 8080
#USER nonroot:nonroot remove for compose
ENTRYPOINT ["/app/app"]
