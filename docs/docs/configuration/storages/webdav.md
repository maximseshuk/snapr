# WebDAV

`type: webdav` — upload to a WebDAV server (Nextcloud, OwnCloud, generic Apache mod_dav, etc.).

## Fields

| Field      | Type     | Required | Notes                             |
| ---------- | -------- | -------- | --------------------------------- |
| `type`     | `webdav` | yes      |                                   |
| `name`     | string   | yes      | identifier; unique within the job |
| `url`      | string   | yes      | base URL of the WebDAV endpoint   |
| `username` | string   | no       |                                   |
| `password` | string   | no       | use `env:`                        |
| `path`     | string   | no       | sub-path inside the endpoint      |

## Example

```yaml
storages:
  - type: webdav
    name: nextcloud
    url: https://cloud.example.com/remote.php/dav/files/backup
    username: backup
    password: env:DAV_PASSWORD
    path: snapr
```
