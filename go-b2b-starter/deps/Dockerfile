FROM golang:1.25.5-alpine

WORKDIR /workspace

# Install system dependencies
RUN apk add --no-cache git make bash curl

# Install Go tools
RUN go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest && \
    go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest && \
    go install github.com/swaggo/swag/cmd/swag@latest && \
    go install github.com/air-verse/air@latest

CMD ["bash"]
