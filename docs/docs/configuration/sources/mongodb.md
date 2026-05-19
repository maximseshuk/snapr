# MongoDB

`type: mongodb` — uses `mongodump` to back up a single database, all databases, or via a connection URI.

## Prerequisites

- `mongodump` must be installed on the snapr host and available in `PATH`.
- `mongodump` is shipped in the **MongoDB Database Tools** package, which is distributed **separately** from the MongoDB server: <https://www.mongodb.com/try/download/database-tools>.
- The configured user needs `backup` role (or equivalent read access on the target database, plus `clusterMonitor` for `--oplog`).

## Fields

| Field           | Type         | Required | Notes                                                                             |
| --------------- | ------------ | -------- | --------------------------------------------------------------------------------- |
| `type`          | `mongodb`    | yes      |                                                                                   |
| `uri`           | string       | yes\*    | full Mongo URI; if set, host/port/credentials below are ignored. Use `env:`.      |
| `host`          | string       | yes\*    | required if `uri` is not set. Defaults to `127.0.0.1`.                            |
| `port`          | int          | no       | default `27017`                                                                   |
| `username`      | string       | no       |                                                                                   |
| `password`      | string       | no       | passed as `--password=...`. Use `env:`. Prefer `uri` to keep secrets off cmdline. |
| `database`      | string       | yes\*    | required unless `allDatabases: true` or `uri` is set                              |
| `allDatabases`  | bool         | no       | omit `--db` so `mongodump` dumps every database                                   |
| `authDatabase`  | string       | no       | `--authenticationDatabase` (typically `admin`)                                    |
| `oplog`         | bool         | no       | include oplog for point-in-time consistency (replica set required)                |
| `excludeTables` | string[]     | no       | passed as `--excludeCollection` flags                                             |
| `extraParams`   | map<str,str> | no       | extra `mongodump` flags                                                           |

## Example — URI

```yaml
sources:
  - type: mongodb
    uri: env:MONGO_URI
    oplog: true
```

## Example — host/port

```yaml
sources:
  - type: mongodb
    host: mongo.internal
    port: 27017
    username: backup
    password: env:MONGO_PASSWORD
    authDatabase: admin
    database: app
    excludeTables:
      - sessions
      - audit
    extraParams:
      gzip: ''
      numParallelCollections: '4'
```

## Notes

- `oplog: true` requires a replica set (or single-node replica set) and is incompatible with `allDatabases`.
- When using `uri`, embed credentials in the URI string itself; the `username`/`password` fields are ignored.
- `extraParams` keys are prefixed with `--`. Use an empty string value for boolean flags.
