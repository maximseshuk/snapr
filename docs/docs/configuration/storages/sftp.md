# SFTP

`type: sftp` — upload to an SFTP server using either password or SSH key auth.

## Fields

| Field            | Type   | Required | Notes                                                                                                          |
| ---------------- | ------ | -------- | -------------------------------------------------------------------------------------------------------------- |
| `type`           | `sftp` | yes      |                                                                                                                |
| `name`           | string | yes      | identifier; unique within the job                                                                              |
| `host`           | string | yes      |                                                                                                                |
| `port`           | int    | no       | default `22`                                                                                                   |
| `username`       | string | no       |                                                                                                                |
| `password`       | string | no       | use `env:`. Can be combined with `privateKey` (both auth methods are offered).                                 |
| `privateKey`     | string | no       | **path** to a PEM private key file on the snapr host. `~` is expanded.                                         |
| `passphrase`     | string | no       | passphrase for the private key. Use `env:`.                                                                    |
| `knownHosts`     | string | no       | path to a `known_hosts` file. Defaults to `~/.ssh/known_hosts`. `~` is expanded.                               |
| `strictHostKey`  | bool   | no       | host-key verification. Defaults to `true` (strict). Set to `false` to disable — **insecure**, accepts any key. |
| `path`           | string | no       | remote directory (created if missing). Defaults to `.`                                                         |
| `includeJobName` | bool   | no       | append the job name under `path` (default `true`); see [Storages](index.md#skipping-the-per-job-subdirectory). |

## Example — password

```yaml
storages:
  - type: sftp
    name: offsite
    host: backup.example.com
    port: 22
    username: backup
    password: env:SFTP_PASSWORD
    path: /uploads
    strictHostKey: true
    knownHosts: /etc/snapr/known_hosts
```

## Example — SSH key

```yaml
storages:
  - type: sftp
    name: offsite-key
    host: backup.example.com
    username: backup
    privateKey: /etc/snapr/keys/sftp_id_ed25519
    passphrase: env:SFTP_PASSPHRASE
    path: /snapr
    strictHostKey: true
    knownHosts: /etc/snapr/known_hosts
```

## Prerequisites

- No external binaries required (snapr uses the Go SSH library).
- The private key file must be readable by the snapr process (mount it into the container if running in Docker).
- For production: pre-populate `knownHosts` with the server's host key (`ssh-keyscan -H <host> >> known_hosts`) and leave `strictHostKey` at the default.

## Notes

- `username` defaults to the OS user running snapr if omitted.
- `port` defaults to `22`.
- When `strictHostKey: false`, snapr logs a warning but accepts any host key. Use only for ad-hoc testing.
