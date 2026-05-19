# Encryption

Optional per-job symmetric encryption applied **after** compression. Currently only OpenSSL is supported.

## Prerequisites

- `openssl` must be installed on the snapr host and available in `PATH`. snapr shells out to it for both encrypt and decrypt operations.
- snapr uses `openssl enc -<cipher> -salt -pbkdf2 -pass env:...`. The password is forwarded through an env var, never on the cmdline.

## Fields

| Field      | Type   | Required | Default       | Notes                                                                           |
| ---------- | ------ | -------- | ------------- | ------------------------------------------------------------------------------- |
| `type`     | enum   | no       | `openssl`     | only `openssl` is supported                                                     |
| `cipher`   | string | no       | `aes-256-cbc` | any cipher accepted by `openssl enc -<cipher>` (e.g. `aes-256-gcm`, `chacha20`) |
| `password` | string | yes      | —             | encryption passphrase. Use `env:` for secrets.                                  |

## Example

```yaml
jobs:
  - name: postgres-nightly
    # ...
    encryption:
      type: openssl
      cipher: aes-256-cbc
      password: env:BACKUP_ENC_PASSWORD
```

The output filename gets a `.enc` suffix appended to the compressed extension (e.g. `backup.tar.gz.enc`).

## Decrypt manually

snapr encrypts with `-pbkdf2`, so decrypt **must** include the same flag:

```bash
openssl enc -d -aes-256-cbc -pbkdf2 \
  -in backup.tar.gz.enc \
  -out backup.tar.gz \
  -pass env:BACKUP_ENC_PASSWORD
```

If you set a non-default `cipher` in the config, use the same value when decrypting.

> Lose the passphrase, lose the backup. Store it somewhere outside snapr — a password manager or a separate secrets store.
