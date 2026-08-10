#!/usr/bin/env bash
# File: scripts/backup_db.sh
# Automated Postgres backup: consistent logical dump -> off-host object
# storage (Cloudflare R2 via AWS CLI) with retention pruning.
#
# Requirements:
#   - pg_dump (PostgreSQL client tools)
#   - aws CLI configured for the R2 endpoint
#
# Environment:
#   PGHOST/PGPORT/PGUSER/PGPASSWORD/PGDATABASE  postgres connection (defaults localhost:5432)
#   R2_ENDPOINT       e.g. https://<accountid>.r2.cloudflarestorage.com
#   R2_ACCESS_KEY     R2 API token access key
#   R2_SECRET_KEY     R2 API token secret
#   R2_BUCKET         bucket name
#   R2_BACKUP_PREFIX  key prefix (default "backups/postgres")
#   BACKUP_RETENTION_DAYS  keep N most recent backups (default 14)
#
# Exit codes: 0 ok, 1 dump/upload failure, 2 missing prerequisites.

set -euo pipefail

PG_HOST="${PGHOST:-localhost}"
PG_PORT="${PGPORT:-5432}"
PG_USER="${PGUSER:-postgres}"
PG_PASS="${PGPASSWORD:-}"
PG_DB="${PGDATABASE:-saas_db}"
R2_ENDPOINT="${R2_ENDPOINT:?R2_ENDPOINT is required}"
R2_ACCESS_KEY="${R2_ACCESS_KEY:?R2_ACCESS_KEY is required}"
R2_SECRET_KEY="${R2_SECRET_KEY:?R2_SECRET_KEY is required}"
R2_BUCKET="${R2_BUCKET:?R2_BUCKET is required}"
R2_BACKUP_PREFIX="${R2_BACKUP_PREFIX:-backups/postgres}"
RETENTION="${BACKUP_RETENTION_DAYS:-14}"

command -v pg_dump >/dev/null 2>&1 || { echo "ERROR: pg_dump not found" >&2; exit 2; }
command -v aws >/dev/null 2>&1 || { echo "ERROR: aws CLI not found" >&2; exit 2; }

export AWS_ACCESS_KEY_ID="$R2_ACCESS_KEY"
export AWS_SECRET_ACCESS_KEY="$R2_SECRET_KEY"
export AWS_DEFAULT_REGION="auto"

STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
KEY="${R2_BACKUP_PREFIX}/${PG_DB}_${STAMP}.dump.gz"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

DUMP_PATH="${TMP_DIR}/backup.dump.gz"

echo "backup_db: dumping ${PG_DB}@${PG_HOST}:${PG_PORT} ..."
if [ -n "$PG_PASS" ]; then
    export PGPASSWORD="$PG_PASS"
fi
pg_dump -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$PG_DB" \
    --format=custom --no-owner --no-acl \
    | gzip -9 > "$DUMP_PATH"

echo "backup_db: computing checksum ..."
sha256sum "$DUMP_PATH" | tee "${DUMP_PATH}.sha256" | cut -d' ' -f1 > "$TMP_DIR/checksum.txt"

SIZE="$(stat -c%s "$DUMP_PATH" 2>/dev/null || stat -f%z "$DUMP_PATH")"
echo "backup_db: dump size ${SIZE} bytes"

echo "backup_db: uploading s3://${R2_BUCKET}/${KEY} ..."
aws s3 cp "$DUMP_PATH" "s3://${R2_BUCKET}/${KEY}" --endpoint-url "$R2_ENDPOINT"
aws s3 cp "${DUMP_PATH}.sha256" "s3://${R2_BUCKET}/${KEY}.sha256" --endpoint-url "$R2_ENDPOINT"

echo "backup_db: pruning backups older than ${RETENTION} days ..."
CUTOFF="$(date -u -d "-${RETENTION} days" +%Y%m%dT%H%M%SZ 2>/dev/null || date -u -v-${RETENTION}d +%Y%m%dT%H%M%SZ)"
aws s3 ls "s3://${R2_BUCKET}/${R2_BACKUP_PREFIX}/" --endpoint-url "$R2_ENDPOINT" \
    | grep "${PG_DB}_" | awk '{print $4}' \
    | while read -r name; do
        stamp="${name#*_}"
        stamp="${stamp%.dump.gz}"
        if [[ "$stamp" < "$CUTOFF" ]]; then
            echo "backup_db: pruning ${name}"
            aws s3 rm "s3://${R2_BUCKET}/${R2_BACKUP_PREFIX}/${name}" --endpoint-url "$R2_ENDPOINT"
            aws s3 rm "s3://${R2_BUCKET}/${R2_BACKUP_PREFIX}/${name}.sha256" --endpoint-url "$R2_ENDPOINT" || true
        fi
    done

echo "backup_db: OK backup=${KEY} size=${SIZE} sha256=$(cat "$TMP_DIR/checksum.txt")"
