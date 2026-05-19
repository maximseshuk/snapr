# Sources

`sources` is an array; a single job may back up multiple sources into one archive. Every entry must set `type` — the rest of the fields depend on the type.

## Supported types

| Type         | Description                                          |
| ------------ | ---------------------------------------------------- |
| `local`      | files and directories on the host                    |
| `sqlite`     | a single SQLite database file                        |
| `postgresql` | PostgreSQL via `pg_dump`                             |
| `mysql`      | MySQL via `mysqldump`                                |
| `mariadb`    | MariaDB via `mariadb-dump` (configured like `mysql`) |
| `mongodb`    | MongoDB via `mongodump`                              |
| `redis`      | Redis snapshot                                       |
| `s3`         | files from an S3-compatible bucket                   |
| `bunny`      | files from a Bunny Storage Zone                      |

Pick a type for the field reference and YAML examples.

## Required external tools

Most database sources shell out to the vendor's own dump utility. snapr **does not bundle** these — they must be present on `PATH` (or inside the snapr Docker image you build/extend).

| Source       | Required binary                            | Typical package (Debian/Ubuntu)                                 |
| ------------ | ------------------------------------------ | --------------------------------------------------------------- |
| `postgresql` | `pg_dump`                                  | `postgresql-client-<version>`                                   |
| `mysql`      | `mysqldump`                                | `mysql-client` or `default-mysql-client`                        |
| `mariadb`    | `mariadb-dump` (falls back to `mysqldump`) | `mariadb-client`                                                |
| `mongodb`    | `mongodump`                                | `mongodb-database-tools` (separate package from MongoDB itself) |
| `redis`      | `redis-cli`                                | `redis-tools`                                                   |
| `sqlite`     | `sqlite3`                                  | `sqlite3`                                                       |
| `local`      | none                                       | —                                                               |
| `s3`         | none (uses AWS SDK)                        | —                                                               |
| `bunny`      | none (uses HTTP API)                       | —                                                               |

If [encryption](/configuration/encryption) is enabled, `openssl` must also be on `PATH`.

> Match the **major version** of `pg_dump` / `mysqldump` to the server version, otherwise the dump may fail or omit data. The official Docker image ships matching client tooling — when running outside Docker, install the client package that pairs with your server.

## How remote sources are fetched

Database, S3, and Bunny sources are **pulled to the snapr host first**, then archived and uploaded to the configured [storages](/configuration/storages/). snapr does not stream object-store data directly between the source and the storage backend.

Practical implications:

- The host needs free disk space at least equal to the (uncompressed) source size, plus headroom for the resulting archive.
- The first run downloads everything. Object-store sources (`s3`, `bunny`) support a persistent local cache via `syncPath` — later runs only fetch files whose size or modification time changed. Without `syncPath` every run re-downloads the full tree.
- For very large buckets factor in egress costs and transfer time.

The working directory is cleaned up after the job finishes (success or failure). When `syncPath` is set, the cache directory is preserved between runs.
