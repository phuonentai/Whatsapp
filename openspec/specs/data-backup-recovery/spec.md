## Purpose

Defines the scheduled Postgres backup regime, the restore procedure, and the RPO/RTO availability contract.

## Requirements

### Requirement: Scheduled Postgres backups

The system SHALL provide an automated backup mechanism that creates Postgres backups on a scheduled cadence and stores them off-host.

#### Scenario: Scheduled backup executes

- **WHEN** the backup schedule triggers
- **THEN** the system SHALL create a consistent logical backup (pg_dump) of the application database
- **AND** SHALL upload the backup artifact to off-host object storage (Cloudflare R2)
- **AND** SHALL record backup success/failure metadata (timestamp, size, checksum) in the webhook log store or a dedicated backup log

#### Scenario: Backup failure is surfaced

- **WHEN** a scheduled backup fails (dump error, upload error, or checksum mismatch)
- **THEN** the failure SHALL be recorded with the error
- **AND** the system SHALL NOT delete or overwrite the previous successful backup

#### Scenario: Backup retention

- **WHEN** backups exceed the configured retention period
- **THEN** the oldest backups SHALL be pruned
- **AND** at least one successful backup SHALL always be retained

### Requirement: Backup restore procedure

The system SHALL document and support restoring the application database from a backup artifact, targeting a defined RPO/RTO contract.

#### Scenario: Restore from backup

- **WHEN** an operator executes the restore procedure with a backup artifact
- **THEN** the system SHALL create a fresh database and load the backup into it
- **AND** the procedure SHALL verify data integrity (row counts / checksums) before the database is accepted

#### Scenario: Restore drill passes

- **WHEN** a restore drill runs against a scratch database
- **THEN** the drill SHALL complete within the RTO target and verify the restored data is queryable
- **AND** the drill result SHALL be recorded

#### Scenario: Restore failure

- **WHEN** a restore from a corrupt or incomplete backup fails
- **THEN** the system SHALL surface the failure
- **AND** SHALL NOT overwrite the live database

### Requirement: RPO/RTO contract

The platform SHALL define and meet a recovery point objective and recovery time objective for the application database.

#### Scenario: Contract defined

- **WHEN** the platform operates in production
- **THEN** the recovery point objective SHALL be at most 24 hours of data (daily backup cadence)
- **AND** the recovery time objective SHALL be at most 4 hours from a full database loss

#### Scenario: Contract violation detection

- **WHEN** a backup has not succeeded within the RPO window
- **THEN** the system SHALL flag the platform as non-compliant with the RPO contract
