# Bunny Storage

`type: bunny` — upload to a [Bunny Storage Zone](https://bunny.net/storage/) using the Bunny HTTP API.

## Prerequisites

- No external binaries required.
- A Storage Zone with read+write API access.
- For signed direct download links (recommended for large archives): a Pull Zone connected to the storage, plus its token authentication key.

## Fields

| Field                  | Type    | Required | Notes                                                                                                          |
| ---------------------- | ------- | -------- | -------------------------------------------------------------------------------------------------------------- |
| `type`                 | `bunny` | yes      |                                                                                                                |
| `name`                 | string  | yes      | identifier; unique within the job                                                                              |
| `endpoint`             | string  | yes      | storage endpoint, e.g. `storage.bunnycdn.com` or a regional variant                                            |
| `zoneName`             | string  | yes      | Storage Zone name                                                                                              |
| `accessKey`            | string  | yes      | Storage Zone password / API key (use `env:`)                                                                   |
| `path`                 | string  | no       | sub-path inside the zone where archives are stored                                                             |
| `includeJobName`       | bool    | no       | append the job name under `path` (default `true`); see [Storages](index.md#skipping-the-per-job-subdirectory). |
| `pullZoneHostname`     | string  | no       | Pull Zone hostname. Required to enable signed direct downloads.                                                |
| `pullZoneTokenAuthKey` | string  | no       | Pull Zone Advanced Token Authentication key (use `env:`). Required to enable signed direct downloads.          |
| `pullZoneTokenTTL`     | int     | no       | token lifetime in seconds. Default `3600`.                                                                     |

## Example

```yaml
storages:
  - type: bunny
    name: bunny-cold
    endpoint: storage.bunnycdn.com
    zoneName: my-zone
    accessKey: env:BUNNY_KEY
    path: snapr/postgres
```

## Download behavior

By default snapr serves downloads by streaming bytes from the Bunny **Storage API** through its own HTTP server. This works without any Pull Zone setup, but the Storage API has tight per-zone connection limits (Bunny throttles aggressive parallel reads with `429 Too Many Requests`).

When both `pullZoneHostname` and `pullZoneTokenAuthKey` are set, snapr instead returns a signed Pull Zone URL and the browser fetches directly from the CDN edge. This is the recommended setup for large archives or busy installations:

```yaml
storages:
  - type: bunny
    name: bunny-cold
    endpoint: storage.bunnycdn.com
    zoneName: my-zone
    accessKey: env:BUNNY_KEY
    path: snapr/postgres
    pullZoneHostname: my-zone.b-cdn.net
    pullZoneTokenAuthKey: env:BUNNY_PULLZONE_KEY
    pullZoneTokenTTL: 1800
```

For split snapshots, signed Pull Zone URLs work for **per-part** downloads but not for the "Download full archive" option — one HTTP redirect cannot represent N parts. The UI hides the full-archive option in that case. Drop the Pull Zone settings (or use a different storage as `defaultStorage`) if full-archive streaming matters more than direct CDN delivery.

## Notes

- The Pull Zone must point at the same Storage Zone you upload to; otherwise signed URLs return 404.
- `pullZoneHostname` accepts both bare hostnames (`my-zone.b-cdn.net`) and `https://...` URLs — snapr strips the scheme.
