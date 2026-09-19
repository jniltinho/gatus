# Installing the binary on Linux with systemd

Gatus is a single static binary: no runtime, no libraries, no container. This page installs the release tarball in
`/opt/gatus` and runs it as a systemd service, under a user of its own and with the file system locked down.

```text
/opt/gatus/gatus                 the binary
/opt/gatus/config/config.yaml    the configuration (several YAML files in this directory are merged)
/opt/gatus/data/                 the SQLite database and nothing else: the only place the service writes
/etc/systemd/system/gatus.service
```

## Install

```bash
VERSION=6.0.2
ARCH=amd64        # or arm64

sudo useradd --system --home-dir /opt/gatus --shell /usr/sbin/nologin gatus
sudo install -d -o root -g gatus -m 0750 /opt/gatus /opt/gatus/config
sudo install -d -o gatus -g gatus -m 0750 /opt/gatus/data

curl -fsSL -o /tmp/gatus.tar.gz \
  "https://github.com/jniltinho/gatus/releases/download/v${VERSION}/gatus_${VERSION}_linux_${ARCH}.tar.gz"
sudo tar xzf /tmp/gatus.tar.gz -C /opt/gatus gatus
sudo chown root:gatus /opt/gatus/gatus && sudo chmod 0750 /opt/gatus/gatus
sudo -u gatus /opt/gatus/gatus version
```

The binary and the configuration belong to `root` and are only readable by the `gatus` group: the service cannot
rewrite its own binary nor its configuration, which holds the password hash and the secrets of the alerts.

## Configure

```bash
sudo -u gatus /opt/gatus/gatus password hash      # asks for the password, prints the hash
sudoedit /opt/gatus/config/config.yaml
```

```yaml
web:
  address: 127.0.0.1      # publish it through nginx or another reverse proxy
  port: 8080
storage:
  type: sqlite
  path: /opt/gatus/data/data.db
security:
  basic:
    username: admin
    password-bcrypt-base64: "JDJhJDEwJD..."
admin:
  enabled: true
endpoints:
  - name: website
    url: "https://example.org"
    interval: 1m
    conditions:
      - "[STATUS] == 200"
```

```bash
sudo chown root:gatus /opt/gatus/config/config.yaml && sudo chmod 0640 /opt/gatus/config/config.yaml
sudo -u gatus /opt/gatus/gatus config validate --config /opt/gatus/config/config.yaml
```

With PostgreSQL, MySQL or MariaDB as the storage, `/opt/gatus/data` stays empty and `storage.path` is the DSN: see
[storage-mysql.md](storage-mysql.md).

## The service

Copy [systemd/gatus.service](systemd/gatus.service) and start it:

```bash
sudo curl -fsSL -o /etc/systemd/system/gatus.service \
  https://raw.githubusercontent.com/jniltinho/gatus/v6.0.2/docs/systemd/gatus.service
sudo systemctl daemon-reload
sudo systemctl enable --now gatus
systemctl status gatus
curl -s http://127.0.0.1:8080/health          # {"status":"UP"}
journalctl -u gatus -f                        # the log, see "Logs" below
```

What the unit does, and why:

| | |
|---|---|
| `ExecStartPre=... config validate` | A restart with an invalid configuration fails before the running process is replaced, with the error in `journalctl`. |
| No `ExecReload` | Gatus reloads the configuration by itself, up to 30 seconds after the file changes. `systemctl restart gatus` is only needed after replacing the binary. Endpoints and status pages created at `/admin` live in the database and need neither. |
| `User=gatus`, `NoNewPrivileges`, `ProtectSystem=strict`, `ReadWritePaths=/opt/gatus/data` | The service runs without root and sees the whole file system as read-only, except its data directory. |
| `AmbientCapabilities=CAP_NET_RAW` | ICMP endpoints (`icmp://host`) open a raw socket, which a user other than root only gets through this capability. Remove the two capability lines if you do not monitor by ping. |
| `RestrictAddressFamilies`, `MemoryDenyWriteExecute`, `SystemCallArchitectures=native`, ... | System call filters. `systemd-analyze security gatus` rates the unit at 3.2 (OK); an unhardened service is around 9.6. |

If an endpoint needs something the sandbox denies — a client certificate in `/home`, an SSH key, a Unix socket — the
check fails with a permission error in the log: grant that path with another `ReadOnlyPaths=` or `ReadWritePaths=`
line in a drop-in (`sudo systemctl edit gatus`) instead of removing the protection.

## Logs

Gatus writes its log to the standard output — it has no log file of its own and no HTTP access log — and systemd sends
it to the journal, under the identifier `gatus`:

```bash
journalctl -u gatus -f                          # follow
journalctl -u gatus --since "1 hour ago"
journalctl -u gatus -b | grep -E "WARN|ERROR"   # since the last boot, only the problems
journalctl -u gatus -o cat | grep "key=core_website"   # one endpoint
```

The level is part of the text of each line, not a priority of the journal, so `journalctl -p warning` does not filter
it: use `grep`. Every line starts with its origin, such as `[watchdog.executeEndpoint]` or `[api.pushHandler]`. Since `v6.0.0` the
lines of the start and of the reload begin with `[cmd.` (before, `[main.`): adjust filters and alerts that match on the
old prefix.

- **Level:** `GATUS_LOG_LEVEL` (`DEBUG`, `INFO`, `WARN`, `ERROR`, `FATAL`), in the unit or in a drop-in
  (`sudo systemctl edit gatus`, then `[Service]` and `Environment=GATUS_LOG_LEVEL=DEBUG`), followed by
  `sudo systemctl restart gatus`. `INFO` logs one line per check, which with many endpoints and short intervals is most
  of the volume; `WARN` keeps only the problems.
- **Keeping the journal across reboots:** on a distribution where `/var/log/journal` does not exist the journal lives in
  memory. `sudo mkdir -p /var/log/journal && sudo systemctl restart systemd-journald` makes it persistent, and
  `SystemMaxUse=1G` in `/etc/systemd/journald.conf` bounds its size.
- **A log file as well**, for a collector that reads files: in a drop-in,

  ```ini
  [Service]
  LogsDirectory=gatus
  StandardOutput=append:/var/log/gatus/gatus.log
  StandardError=inherit
  ```

  systemd creates `/var/log/gatus` for the `gatus` user and opens the file itself, so the sandbox needs no other
  change. Rotate it with `copytruncate`, because the file stays open while the service runs:

  ```text
  # /etc/logrotate.d/gatus
  /var/log/gatus/gatus.log {
      daily
      rotate 14
      compress
      missingok
      notifempty
      copytruncate
  }
  ```

## Upgrade

```bash
VERSION=x.y.z; ARCH=amd64     # the new version
curl -fsSL -o /tmp/gatus.tar.gz \
  "https://github.com/jniltinho/gatus/releases/download/v${VERSION}/gatus_${VERSION}_linux_${ARCH}.tar.gz"
sudo cp -a /opt/gatus/gatus /opt/gatus/gatus.previous
sudo tar xzf /tmp/gatus.tar.gz -C /opt/gatus gatus
sudo chown root:gatus /opt/gatus/gatus && sudo chmod 0750 /opt/gatus/gatus
sudo systemctl restart gatus && systemctl status gatus
```

The database is migrated when the new version starts. To go back, restore `gatus.previous` and restart; before an
upgrade that skips several versions, download a backup at `/admin/backup` first
(see [admin-endpoints.md](admin-endpoints.md#backup-and-restore)).

## Behind nginx

```nginx
server {
    listen 443 ssl;
    server_name status.example.com;
    # ssl_certificate ...;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    # Real-time updates are an event stream: it must not be buffered, and it stays open for minutes
    location ~ ^/api/v1/(endpoints|status-pages)/.*/events$ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_buffering off;
        proxy_read_timeout 7m;
    }
}
```

`Host` and `X-Forwarded-Proto` are what the administration uses to accept a change as coming from its own address, and
what marks the session cookie as `Secure`: see [admin-endpoints.md](admin-endpoints.md#behind-a-reverse-proxy).

## Remove

```bash
sudo systemctl disable --now gatus
sudo rm /etc/systemd/system/gatus.service && sudo systemctl daemon-reload
sudo rm -rf /opt/gatus          # includes the database
sudo userdel gatus
```
