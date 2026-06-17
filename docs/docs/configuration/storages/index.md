# Storages

`storages` is an array of destinations. A job uploads each archive to **all** configured storages. Use `defaultStorage` on the job to mark which one the UI surfaces first.

## Supported types

| Type     | Description                         |
| -------- | ----------------------------------- |
| `local`  | local filesystem path               |
| `s3`     | AWS S3 or any S3-compatible service |
| `sftp`   | SFTP server (password or SSH key)   |
| `webdav` | WebDAV server                       |
| `bunny`  | Bunny Storage Zone                  |

Every storage entry **must** set a `name`. It identifies the storage in `defaultStorage`, in logs, and in the UI. Names must be unique within a single job.

```yaml
storages:
  - name: local-disk
    type: local
    path: /var/backups
  - name: offsite
    type: s3
    # ...
defaultStorage: offsite
```

## On-disk layout

snapr writes every snapshot under a per-job subdirectory inside the storage's `path`:

```
<storage.path>/
  <job-name>/
    <job-name>-20260509-030000.tar.gz
    <job-name>-20260510-030000.tar.gz.enc
```

This layout is the same on every backend (Local, S3, SFTP, WebDAV, Bunny). It lets snapr filter by job natively — S3 uses `Prefix`, the others list one directory — instead of scanning the whole `path` and filtering by name. Sharing one storage across many jobs is cheap.

Split snapshots get an extra wrapper directory; see [Splitter](../splitter.md).

### Skipping the per-job subdirectory

Set `includeJobName: false` on a storage to drop the `<job-name>/` level and write snapshots directly under `path`:

```yaml
storages:
  - name: offsite
    type: s3
    bucket: backups
    path: db
    includeJobName: false
```

```
db/
  myjob-20260509-030000.tar.gz
```

`includeJobName` is optional and defaults to `true`, so existing configs keep the per-job subdirectory. It is supported by every backend. Only set it to `false` when a single job owns the `path` — sharing one `path` across jobs without the per-job level mixes their snapshots together.

## Listing cache

The backups list returned to the UI is cached per `(job, storage)` pair for **5 minutes**. snapr invalidates the cache immediately after every upload, delete, and retention sweep, so changes made by snapr itself are visible right away. Files placed or removed manually (outside of snapr) appear up to 5 minutes later.
