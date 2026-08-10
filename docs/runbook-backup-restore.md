# Runbook: Postgres Backup & Restore

Owner: Platform / Ops. Contract: **RPO ≤ 24h**, **RTO ≤ 4h** (see `openspec/specs/data-backup-recovery`).

## Backup

Automatic: host cron (or container job) runs `go-b2b-starter/scripts/backup_db.sh` daily.

```cron
0 2 * * * cd /opt/platform/go-b2b-starter && ./scripts/backup_db.sh >> /var/log/backup_db.log 2>&1
```

Required env (secrets via your env store, never commit):

| Variable | Example |
|----------|---------|
| `PGHOST`/`PGPORT`/`PGUSER`/`PGPASSWORD`/`PGDATABASE` | prod connection |
| `R2_ENDPOINT` | `https://<account>.r2.cloudflarestorage.com` |
| `R2_ACCESS_KEY` / `R2_SECRET_KEY` | R2 token with write on the backup prefix |
| `R2_BUCKET` | `platform-backups` |
| `BACKUP_RETENTION_DAYS` | `14` |

Artifact layout: `backups/postgres/<db>_<timestamp>.dump.gz` + `.sha256`.

**Failure signal:** script exits non-zero; log line `ERROR:`. The previous successful backup is never deleted on failure. If a backup has not succeeded within the RPO window, the platform is non-compliant — investigate immediately (disk, R2 token, postgres down).

## Restore

```bash
cd /opt/platform/go-b2b-starter
RESTORE_DB=saas_db_restore \
BACKUP_FILE=saas_db_20260810T020000Z.dump.gz \
./scripts/restore_db.sh
```

- Restores into a **fresh** database. The live DB is never overwritten.
- Verifies SHA-256 and organization row count; prints the count for manual confirmation.
- On restore failure the partial database is dropped.

After verification, switch the application to the restored DB (update `POSTGRES_DB` env + restart backend), or use `pg_dump | pg_restore` to move data into the live DB deliberately.

## RPO / RTO

- RPO: ≤ 24h (daily cadence; logical dump, no PITR).
- RTO: ≤ 4h from full database loss (download artifact → restore → verify → repoint app).
- Restore drills: run quarterly in staging; results recorded in this runbook.

## Tested Restore Drill (record)

| Date | Artifact | Duration | Result |
|------|----------|----------|--------|
| (CI) | scratch dump | < 10 min | automated |
