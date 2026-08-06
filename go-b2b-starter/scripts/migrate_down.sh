#!/bin/bash

# Load environment variables from a file if it exists
ENV_FILE="app.env"
if [ -f "$ENV_FILE" ]; then
    source "$ENV_FILE"
else
    echo "Environment file not found, ensure $ENV_FILE exists or set the variables manually."
    exit 1
fi

# Define migration paths
MIGRATION_PATHS=(
    "src/pkg/db/postgres/sqlc/migrations"

)

# Perform migrations
for path in "${MIGRATION_PATHS[@]}"; do
    echo "Migrating up in $path..."
    migrate -path $path -database "postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable" -verbose down
    if [ $? -ne 0 ]; then
        echo "Migration failed for $path"
        exit 1
    else
        echo "Migration completed for $path"
    fi
done
