# MariaDB

`type: mariadb` — produces a logical dump using `mariadb-dump` when available, falling back to `mysqldump`.

The configuration is identical to [MySQL](/configuration/sources/mysql) — both tools accept the same flags. snapr's source picks the binary at runtime; you do not select it in the config.

## Prerequisites

- Either `mariadb-dump` (preferred) or `mysqldump` must be installed on the snapr host and available in `PATH`.
- Install via the `mariadb-client` package on Debian/Ubuntu (provides `mariadb-dump`), or `mysql-client` for the fallback.
- The backup user needs the same grants as MySQL: at minimum `SELECT`, `LOCK TABLES`, `SHOW VIEW`, `EVENT`, `TRIGGER`, `PROCESS`.

## Fields

Same as [MySQL](/configuration/sources/mysql#fields). All options apply.

## Example

```yaml
sources:
  - type: mariadb
    host: db.internal
    port: 3306
    username: backup
    password: env:MARIADB_PASSWORD
    database: app
    extraParams:
      single-transaction: ''
```

## Example — all databases

```yaml
sources:
  - type: mariadb
    host: db.internal
    username: backup
    password: env:MARIADB_PASSWORD
    allDatabases: true
```

## Notes

- snapr first looks up `mariadb-dump` in `PATH`; if missing, it falls back to `mysqldump`. Install at least one.
- For mixed MySQL/MariaDB hosts, prefer the matching client package — flag compatibility is good but not perfect across major versions.
