#!/usr/bin/env bash
# File: scripts/restore_db.sh
# Restore a Postgres backup artifact into a FRESH database and verify it.
# The live database is NEVER overwritten; restore always targets a new DB.
#
# Environment:
#   PGHOST/PGPORT/PGUSER/PGPASSWORD  postgres superuser connection
#   RESTORE_DB          name of the fresh database to create (default saas_db_restore)
#   SOURCE_DB           name of the source database the backup was taken from
#                       (default saas_db; used to find the artifact and verify)
#   R2_ENDPOINT / R2_ACCESS_KEY / R2_SECRET_KEY / R2_BUCKET   artifact store
#   R2_BACKUP_PREFIX    key prefix (default backups/postgres)
#   BACKUP_FILE         optional explicit artifact name; else latest is used
#   EXPECT_ORGS         optional expected count for organizations.organizations

set -euo pipefail

PG_HOST="${PGHOST:-localhost}"
PG_PORT="${PGPORT:-5432}"
PG_USER="${PGUSER:-postgres}"
PG_PASS="${PGPASSWORD:-}"
RESTORE_DB="${RESTORE_DB:-saas_db_restore}"
SOURCE_DB="${SOURCE_DB:-saas_db}"
R2_ENDPOINT="${R2_ENDPOINT:?R2_ENDPOINT is required}"
R2_ACCESS_KEY="${R2_ACCESS_KEY:?R2_ACCESS_KEY is required}"
R2_SECRET_KEY="${R2_SECRET_KEY:?R2_SECRET_KEY is required}"
R2_BUCKET="${R2_BUCKET:?R2_BUCKET is required}"
R2_BACKUP_PREFIX="${R2_BACKUP_PREFIX:-backups/postgres}"

command -v pg_restore >/dev/null 2>&1 || { echo "ERROR: pg_restore not found" >&2; exit 2; }
command -v aws >/dev/null 2>&1 || { echo "ERROR: aws CLI not found" >&2; exit 2; }
command -v psql >/dev/null 2>&1 || { echo "ERROR: psql not found" >&2; exit 2; }

export AWS_ACCESS_KEY_ID="$R2_ACCESS_KEY"
export AWS_SECRET_ACCESS_KEY="$R2_SECRET_KEY"
export AWS_DEFAULT_REGION="auto"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

if [ -n "$PG_PASS" ]; then
    export PGPASSWORD="$PG_PASS"
fi

if [ -z "${BACKUP_FILE:-}" ]; then
    BACKUP_FILE="$(aws s3 ls "s3://${R2_BUCKET}/${R2_BACKUP_PREFIX}/" --endpoint-url "$R2_ENDPOINT" \
        | grep "${SOURCE_DB}_" | grep -v sha256 | awk '{print $4}' | sort | tail -1)"
fi
if [ -z "$BACKUP_FILE" ]; then
    echo "ERROR: no backup artifact found for ${SOURCE_DB}" >&2
    exit 2
fi

echo "restore_db: downloading ${BACKUP_FILE} ..."
aws s3 cp "s3://${R2_BUCKET}/${R2_BACKUP_PREFIX}/${BACKUP_FILE}" "$TMP_DIR/backup.dump.gz" --endpoint-url "$R2_ENDPOINT"
aws s3 cp "s3://${R2_BUCKET}/${R2_BACKUP_PREFIX}/${BACKUP_FILE}.sha256" "$TMP_DIR/backup.dump.gz.sha256" --endpoint-url "$R2_ENDPOINT" || true

if [ -f "$TMP_DIR/backup.dump.gz.sha256" ]; then
    echo "restore_db: verifying checksum ..."
    (cd "$TMP_DIR" && sha256sum -c backup.dump.gz.sha256)
fi

echo "restore_db: creating fresh database ${RESTORE_DB} ..."
psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d postgres \
    -c "DROP DATABASE IF EXISTS ${RESTORE_DB};" \
    -c "CREATE DATABASE ${RESTORE_DB};"

echo "restore_db: restoring ..."
gzip -dc "$TMP_DIR/backup.dump.gz" > "$TMP_DIR/backup.dump"
if ! pg_restore -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$RESTORE_DB" \
        --no-owner --no-acl --exit-on-error "$TMP_DIR/backup.dump"; then
    echo "ERROR: restore failed; dropping partial database ${RESTORE_DB}" >&2
    psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d postgres -c "DROP DATABASE IF EXISTS ${RESTORE_DB};" || true
    exit 1
fi

echo "restore_db: verifying restored data ..."
ORG_COUNT="$(psql -h "$PG_HOST" -p "$PG_PORT" -U "$PG_USER" -d "$RESTORE_DB" -tA -c "SELECT COUNT(*) FROM organizations.organizations;")"
echo "restore_db: organizations count = ${ORG_COUNT}"
if [ -n "${EXPECT_ORGS:-}" ] && [ "$ORG_COUNT" != "$EXPECT_ORGS" ]; then
    echo "ERROR: expected ${EXPECT_ORGS} organizations, got ${ORG_COUNT}" >&2
    exit 1
fi

echo "restore_db: OK database ${RESTORE_DB} restored and verified"
