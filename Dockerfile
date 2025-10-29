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
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app ./cmd/server

############################
# Runtime (lille image)
############################
FROM gcr.io/distroless/static:nonroot
WORKDIR /app

# Binær + assets
COPY --from=build /app/app /app/app
COPY --from=build /app/templates /app/templates
COPY --from=build /app/static /app/static

# Demo: tag din SQLite-db med i imaget
# (Hvis jeres lærer hellere vil have ekstern DB, kan vi ændre det senere)
COPY internal/db/whoknows.db /app/whoknows.db

# Miljøvariabler (din main.go læser disse)
ENV PORT=8080
ENV DATABASE_PATH=/app/whoknows.db


EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/app"]