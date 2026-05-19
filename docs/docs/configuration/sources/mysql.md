# MySQL

`type: mysql` — uses `mysqldump` to produce a logical dump.

For MariaDB use [`type: mariadb`](/configuration/sources/mariadb) instead — it prefers `mariadb-dump` and falls back to `mysqldump`.

## Prerequisites

- `mysqldump` must be installed on the snapr host and available in `PATH`.
- Install via your distro's `mysql-client` / `default-mysql-client` package.
- The backup user typically needs at minimum `SELECT`, `LOCK TABLES`, `SHOW VIEW`, `EVENT`, `TRIGGER`, and (when dumping replicas / using `--single-transaction`) `PROCESS`.

## Fields

| Field           | Type         | Required | Notes                                                                   |
| --------------- | ------------ | -------- | ----------------------------------------------------------------------- |
| `type`          | `mysql`      | yes      |                                                                         |
| `host`          | string       | yes\*    | required unless `socket` is set. Defaults to `127.0.0.1` if both unset. |
| `port`          | int          | no       | default `3306`                                                          |
| `socket`        | string       | no       | UNIX socket path (alternative to host/port)                             |
| `username`      | string       | no       | passed as `-u`                                                          |
| `password`      | string       | no       | passed via `MYSQL_PWD` env var, never on cmdline. Use `env:`            |
| `database`      | string       | yes\*    | required unless `allDatabases: true`                                    |
| `allDatabases`  | bool         | no       | dump every database on the server (`--all-databases`)                   |
| `tables`        | string[]     | no       | only dump these tables from `database`                                  |
| `excludeTables` | string[]     | no       | passed as `--ignore-table=<db>.<table>` (single-database mode only)     |
| `extraParams`   | map<str,str> | no       | extra `mysqldump` flags                                                 |

## Example — single database

```yaml
sources:
  - type: mysql
    host: db.internal
    port: 3306
    username: backup
    password: env:MYSQL_PASSWORD
    database: app
    excludeTables:
      - sessions
    extraParams:
      single-transaction: ''
      quick: ''
```

## Example — all databases via socket

```yaml
sources:
  - type: mysql
    socket: /var/run/mysqld/mysqld.sock
    username: backup
    password: env:MYSQL_PASSWORD
    allDatabases: true
```

## Notes

- `extraParams` keys are prefixed with `--`. Use an empty string value for boolean flags.
- `excludeTables` is ignored when `allDatabases: true` is set.
