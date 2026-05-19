# Splitter

`split` cuts the final archive into fixed-size parts before upload. snapr applies it **after** compression and (if configured) encryption, so each part is a slice of the encrypted/compressed bytes — not a self-contained mini-archive.

Use this when:

- a single storage backend rejects objects above some size (FTP/SFTP server quotas, e-mail attachment limits, providers without S3 multipart upload),
- you want backups to fit on removable media of a known capacity,
- a flaky network would benefit from re-uploading individual parts on failure rather than the whole archive.

## Fields

| Field       | Type   | Required | Notes                                                                                                                                                                                 |
| ----------- | ------ | -------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `chunkSize` | string | yes      | size per part. Accepts `B`, `KB`/`KiB`, `MB`/`MiB`, `GB`/`GiB`, `TB`/`TiB`. Both binary and decimal suffixes are interpreted as powers of 1024 (so `1MB` = `1MiB` = 1 048 576 bytes). |

## Example

```yaml
jobs:
  - name: huge-postgres
    schedule: '0 3 * * *'
    compression: tar.gz
    sources:
      - type: postgresql
        host: db.internal
        username: postgres
        password: env:PG_PASSWORD
        database: app
    storages:
      - name: offsite
        type: sftp
        host: backup.example.com
        username: backup
        privateKey: /etc/snapr/keys/sftp
        path: /snapr
    retention:
      last: 14
    encryption:
      type: openssl
      password: env:BACKUP_ENC_PASSWORD
    split:
      chunkSize: 1GB
```

## On-disk layout

A split snapshot lives in a wrapper directory inside the [per-job folder](storages/index.md#on-disk-layout). Parts are real files inside it:

```
<storage.path>/huge-postgres/
  huge-postgres-20260507-030000.tar.gz.enc.parts-3-3221225472/
    huge-postgres-20260507-030000.tar.gz.enc.part-aaa
    huge-postgres-20260507-030000.tar.gz.enc.part-aab
    huge-postgres-20260507-030000.tar.gz.enc.part-aac
```

The wrapper name encodes both the **part count** and the **total size in bytes** (`.parts-3-3221225472` = 3 parts, ≈3 GiB). snapr reads these from the directory name alone, so the backups listing never has to descend into the wrapper or stat individual parts. Listing one job's backups is one storage call regardless of how many parts each split snapshot has.

The wrapper is created atomically at upload time. snapr never modifies it later — retention either keeps the whole wrapper or deletes it entirely.

## How parts are named

Each part gets a 3-letter alphabetic suffix `aaa` … `zzz`, giving a hard limit of **17 576 parts per archive**. With a 1 GiB chunk that's a 17.1 TiB archive ceiling — increase `chunkSize` if you ever come close.

## Retention treats a set as one backup

snapr treats one wrapper directory as one logical backup. `retention.last: 14` keeps the **14 newest snapshots**, regardless of how many parts each one produced. Rotation removes the wrapper and all its parts in one operation.

## Downloading a split backup

The web UI and `GET /api/v1/jobs/{name}/backups/{filename}/download` accept the **set ID** (the original archive name without the wrapper suffix and without `.part-XXX`). snapr opens each part on the storage, streams them back-to-back as one continuous body, and serves the result as a single download. The client gets one file — no manual concatenation needed.

The UI also exposes per-part download for split snapshots, useful when one network transfer of the full archive is impractical.

For Bunny Storage configured with a signed Pull Zone, full-set download is not supported — one HTTP redirect cannot represent N parts. The UI's "Download all" returns `501 Not Implemented` in this case. Per-part download still works (each part is its own redirect to a signed URL). For full-set streaming use a non-redirect-based storage (Local, S3, SFTP, WebDAV) as the default download source.

## Reassembling parts manually

If you copy a wrapper directory off the storage yourself, concatenate the parts in lexicographic order from inside the wrapper:

```bash
cd huge-postgres-20260507-030000.tar.gz.enc.parts-3-3221225472
cat *.part-* > ../archive.tar.gz.enc
cd ..
openssl enc -d -aes-256-cbc -pbkdf2 \
  -in archive.tar.gz.enc \
  -out archive.tar.gz \
  -pass env:BACKUP_ENC_PASSWORD
tar -xzf archive.tar.gz
```

The shell glob expands in lexicographic order, which matches the order parts were written.

## Toggling split on or off

Adding or removing the `split` block between runs is safe. Existing snapshots keep their original layout — wrapper directories from past split runs and plain archives from non-split runs coexist in the same job folder. Listing, retention, and download handle both transparently.

## Notes

- `chunkSize` is parsed once at config load. An unparseable value fails startup with a validation error.
- Split runs purely in Go — no external `split` binary required.
- A part smaller than `chunkSize` is always the **last** part. Don't use the size of an arbitrary part to estimate `chunkSize`.
- If the source archive fits in a single chunk, snapr writes it directly into the job folder without a wrapper — `partsCount: 1` is never produced.
