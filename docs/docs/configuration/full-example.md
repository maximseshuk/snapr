# Full example

A complete `snapr.yaml` exercising the most common features. A working dev variant lives at [`examples/dev/snapr.yaml`](https://github.com/maximseshuk/snapr/blob/main/examples/dev/snapr.yaml).

```yaml
logs:
  path: /var/log/snapr
  system: true
  perJob: true
  maxSizeMB: 100
  maxBackups: 7
  maxAgeDays: 30
  compress: true

server:
  enabled: true
  address: '0.0.0.0:8080'
  secret: env:SNAPR_JWT_SECRET
  defaultLanguage: en
  auth:
    enabled: true
    username: admin
    password: env:SNAPR_ADMIN_PASSWORD
    tokenExpiration: 60
  permissions:
    allowBackupDownload: true
    allowManualRun: true
    showConfig: true
  ui:
    enabled: true

jobs:
  - name: files-to-s3
    schedule: '0 2 * * *'
    compression: tar.gz
    sources:
      - type: local
        path: /var/data
        excludes: ['*.tmp', '*.log']
    storages:
      - name: s3-primary
        type: s3
        bucket: backups
        region: us-east-1
        accessKeyId: env:S3_KEY
        secretAccessKey: env:S3_SECRET
        storageClass: STANDARD_IA
    retention:
      last: 7

  - name: postgres-nightly
    schedule: '0 3 * * *'
    compression: tar.gz
    sources:
      - type: postgresql
        host: db.internal
        port: 5432
        username: postgres
        password: env:PG_PASSWORD
        database: app
    storages:
      - name: local-disk
        type: local
        path: /var/backups
      - name: offsite-sftp
        type: sftp
        host: offsite.example.com
        username: backup
        privateKey: /etc/snapr/keys/sftp_id_ed25519
        path: /snapr
    retention:
      last: 30
    encryption:
      type: openssl
      cipher: aes-256-cbc
      password: env:BACKUP_ENC_PASSWORD
    notifiers:
      - type: telegram
        botToken: env:TG_BOT_TOKEN
        chatId: '-1001234567890'
        onFailure: true

  - name: huge-archive-split
    schedule: '0 4 * * 0'
    compression: tar.gz
    sources:
      - type: local
        path: /var/lib/bigdata
    storages:
      - name: offsite-sftp
        type: sftp
        host: offsite.example.com
        username: backup
        privateKey: /etc/snapr/keys/sftp_id_ed25519
        path: /snapr/big
    retention:
      last: 4
    encryption:
      type: openssl
      password: env:BACKUP_ENC_PASSWORD
    split:
      chunkSize: 1GB
```
