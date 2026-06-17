# Local

`type: local` — write the archive to a directory on the host (or a mounted volume in Docker).

## Fields

| Field            | Type    | Required | Notes                                                                                                           |
| ---------------- | ------- | -------- | --------------------------------------------------------------------------------------------------------------- |
| `type`           | `local` | yes      |                                                                                                                 |
| `name`           | string  | yes      | identifier; unique within the job                                                                               |
| `path`           | string  | yes      | absolute path; must be writable                                                                                 |
| `includeJobName` | bool    | no       | append the job name under `path` (default `true`); see [Storages](./index.md#skipping-the-per-job-subdirectory) |

## Example

```yaml
storages:
  - type: local
    name: nas
    path: /mnt/nas/snapr
```
