# syntax=docker/dockerfile:1

############################
# Builder stage
############################
FROM golang:1.25 AS build
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app ./cmd/server


############################
# Runtime stage
############################
FROM alpine:3.20
WORKDIR /app

# Create non-root user and group for runtime security
RUN addgroup -S app && adduser -S app -G app

# Use non-root user for the application
USER app

COPY --from=build /app/app .

COPY --from=build /app/templates ./templates
COPY --from=build /app/static ./static
COPY --from=build /app/internal ./internal
COPY --from=build /app/handlers ./handlers
COPY --from=build /app/migrations ./migrations
COPY --from=build /app/scripts ./scripts

ENV PORT=8080

# DATABASE_URL must be provided via docker-compose or k8s
EXPOSE 8080

ENTRYPOINT ["./app"]
