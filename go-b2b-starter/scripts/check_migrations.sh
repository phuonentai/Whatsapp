#!/bin/bash

# File: scripts/check_migrations.sh
# Fails if any migration version prefix maps to more than one migration-set
# base name (i.e., duplicate version numbers). Up/down pairs are legal.

set -euo pipefail

MIGRATIONS_DIR="${MIGRATIONS_DIR:-internal/db/postgres/sqlc/migrations}"

if [ ! -d "$MIGRATIONS_DIR" ]; then
    echo "ERROR: migrations directory not found: $MIGRATIONS_DIR" >&2
    exit 1
fi

cd "$MIGRATIONS_DIR"

fail=0
for v in $(ls *.sql 2>/dev/null | grep -oE '^[0-9]+' | sort -u); do
    sets=$(ls ${v}_*.sql 2>/dev/null | sed 's/^[0-9]*_//; s/\.\(up\|down\)\.sql$//' | sort -u)
    count=$(printf '%s\n' "$sets" | sed '/^$/d' | wc -l)
    if [ "$count" -gt 1 ]; then
        echo "ERROR: duplicate migration version ${v}: $(printf '%s' "$sets" | tr '\n' ' ')" >&2
        fail=1
    fi
done

if [ "$fail" -ne 0 ]; then
    echo "Migration version check FAILED" >&2
    exit 1
fi

echo "OK: all migration versions are unique"
