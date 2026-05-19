# Jobs

`jobs` is an array of backup jobs. Each job has a cron schedule, one or more sources, one or more storages, and a retention policy.

## Fields

| Field            | Type   | Required | Notes                                                                                   |
| ---------------- | ------ | -------- | --------------------------------------------------------------------------------------- |
| `name`           | string | yes      | unique job name                                                                         |
| `schedule`       | cron   | yes      | 5-field cron — see [Schedule](#schedule)                                                |
| `compression`    | enum   | no       | `tar`, `tar.gz`, `tar.zst`, `tar.xz`, `zip` — see [Compression](#compression)           |
| `sources`        | array  | yes      | at least one — see [Sources](/configuration/sources/)                                   |
| `storages`       | array  | yes      | at least one — see [Storages](/configuration/storages/)                                 |
| `defaultStorage` | string | no       | `name` of the storage flagged as default in the UI                                      |
| `retention`      | object | yes      | see [Retention](#retention)                                                             |
| `beforeScript`   | string | no       | shell snippet executed before the job — see [Hook scripts](#hook-scripts)               |
| `afterScript`    | string | no       | shell snippet executed after the job — see [Hook scripts](#hook-scripts)                |
| `encryption`     | object | no       | symmetric encryption — see [Encryption](/configuration/encryption)                      |
| `split`          | object | no       | split the final archive into fixed-size parts — see [Splitter](/configuration/splitter) |
| `notifiers`      | array  | no       | per-job notifications — see [Notifiers](/configuration/notifiers/)                      |

## Schedule

Standard 5-field cron — `minute hour day-of-month month day-of-week`.

| Expression     | Meaning                       |
| -------------- | ----------------------------- |
| `0 2 * * *`    | every day at 02:00            |
| `*/15 * * * *` | every 15 minutes              |
| `0 3 * * 0`    | every Sunday at 03:00         |
| `30 4 1 * *`   | 04:30 on the 1st of the month |
| `0 0 * * 1-5`  | midnight on weekdays          |

Even with a future-dated cron, a job can be triggered immediately from the UI (**Run now**) or via the REST API. Manual runs respect retention and notifiers exactly like scheduled runs. To disable manual runs, set [`server.permissions.allowManualRun: false`](/configuration/server/permissions).

## Compression

| Value     | Output     | Parallel | Notes                                                |
| --------- | ---------- | -------- | ---------------------------------------------------- |
| `tar`     | `.tar`     | —        | no compression — use for already-compressed payloads |
| `tar.gz`  | `.tar.gz`  | yes¹     | classic gzip — best compatibility                    |
| `tar.zst` | `.tar.zst` | yes      | zstd with `-T0` — best speed-vs-ratio trade-off      |
| `tar.xz`  | `.tar.xz`  | yes      | xz with `-T0` — highest ratio, slowest               |
| `zip`     | `.zip`     | —        | zip archive — handy for Windows recipients           |

¹ `tar.gz` uses [pigz](https://zlib.net/pigz/) when it's on `PATH` (all cores); otherwise it falls back to single-threaded `gzip` transparently.

Aliases: `gz` → `tar.gz`, `zst` / `zstd` → `tar.zst`, `xz` → `tar.xz`. Omit `compression` to use plain `tar` (no compression).

The official Docker image ships `tar`, `pigz`, `zstd`, `xz`, and `zip` preinstalled. On bare metal, install whichever compressor matches the formats you use — snapr returns a clear error at run time if a required binary is missing.

## Retention

`retention.last` is **required** and must be at least `1`. After every successful run snapr keeps the N newest archives for that job and deletes the rest from every storage attached to the job.

| Field  | Type | Required | Notes                            |
| ------ | ---- | -------- | -------------------------------- |
| `last` | int  | yes      | number of archives to keep (≥ 1) |

```yaml
retention:
  last: 30
```

## Hook scripts

`beforeScript` and `afterScript` run arbitrary shell commands around the job — quiesce a service, flush caches, post-clean. Both fields are optional and accept multi-line YAML strings.

```yaml
beforeScript: |
  systemctl stop my-app
afterScript: |
  systemctl start my-app
```

`afterScript` runs whether the job succeeded or failed.

## Example

```yaml
jobs:
  - name: postgres-nightly
    schedule: '0 3 * * *'
    compression: tar.gz
    sources:
      - type: postgresql
        host: db.internal
        username: postgres
        password: env:PG_PASSWORD
        database: app
    storages:
      - name: nas
        type: local
        path: /var/backups
      - name: offsite
        type: s3
        bucket: backups
        region: us-east-1
        accessKeyId: env:S3_KEY
        secretAccessKey: env:S3_SECRET
    defaultStorage: offsite
    retention:
      last: 30
    beforeScript: |
      systemctl stop my-app
    afterScript: |
      systemctl start my-app
    encryption:
      type: openssl
      cipher: aes-256-cbc
      password: env:BACKUP_ENC_PASSWORD
```
