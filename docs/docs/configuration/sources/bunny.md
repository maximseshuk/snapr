# Bunny Storage (source)

`type: bunny` — pulls files from a [Bunny Storage Zone](https://bunny.net/storage/) using the Bunny HTTP API.

## Prerequisites

- No external binaries required.
- A Storage Zone access password (FTP/API password). Read-only is sufficient.
- For private files served via a Pull Zone, optionally a Pull Zone token auth key.

## Fields

| Field                  | Type     | Required | Notes                                                                                                                                          |
| ---------------------- | -------- | -------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| `type`                 | `bunny`  | yes      |                                                                                                                                                |
| `endpoint`             | string   | yes      | storage endpoint, e.g. `storage.bunnycdn.com` or a regional variant                                                                            |
| `zoneName`             | string   | yes      | Storage Zone name                                                                                                                              |
| `accessKey`            | string   | yes      | Storage Zone password / API key (use `env:`)                                                                                                   |
| `path`                 | string   | no       | remote sub-path inside the zone to start scanning from (default: zone root)                                                                    |
| `syncPath`             | string   | no       | **local** directory used as a persistent cache. When set, snapr keeps already-downloaded files between runs and only fetches new/changed ones. |
| `excludes`             | string[] | no       | glob patterns matched against zone-relative paths (skipped)                                                                                    |
| `extraParams.workers`  | string   | no       | parallel download workers (default `10`, max `30`)                                                                                             |
| `pullZoneHostname`     | string   | no       | optional Pull Zone hostname for token-authenticated downloads                                                                                  |
| `pullZoneTokenAuthKey` | string   | no       | Pull Zone token auth key                                                                                                                       |
| `pullZoneTokenTTL`     | int      | no       | token TTL in seconds                                                                                                                           |

## How it works

snapr lists the Storage Zone (recursively under `path` if set) and downloads each file. Without `syncPath` it uses a temporary directory cleaned up after the run. With `syncPath` it reuses the directory across runs and skips files whose size + LastModified already match locally — turning subsequent backups into incremental fetches.

The downloaded tree is then compressed, encrypted, and uploaded to the configured [storages](/configuration/storages/). There is no direct zone-to-storage transfer. See [Sources → How remote sources are fetched](/configuration/sources/#how-remote-sources-are-fetched).

## Example

```yaml
sources:
  - type: bunny
    endpoint: storage.bunnycdn.com
    zoneName: my-zone
    accessKey: env:BUNNY_KEY
    path: /uploads
    excludes:
      - '*.tmp'
```

## Example — incremental sync with persistent cache

```yaml
sources:
  - type: bunny
    endpoint: storage.bunnycdn.com
    zoneName: my-zone
    accessKey: env:BUNNY_KEY
    syncPath: /var/cache/snapr/bunny
    extraParams:
      workers: '20'
```
