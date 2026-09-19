<a href="https://github.com/jniltinho/gatus">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset=".github/assets/logo-with-light-text.png">
    <img alt="Gatus" src=".github/assets/logo-with-dark-text.png" width="420">
  </picture>
</a>

# Gatus (jniltinho/gatus)

[![Release](https://img.shields.io/github/v/release/jniltinho/gatus?sort=semver)](https://github.com/jniltinho/gatus/releases)
[![CI](https://github.com/jniltinho/gatus/actions/workflows/ci.yml/badge.svg)](https://github.com/jniltinho/gatus/actions/workflows/ci.yml)
[![Docker pulls](https://img.shields.io/docker/pulls/jniltinho/gatus)](https://hub.docker.com/r/jniltinho/gatus)
[![License](https://img.shields.io/github/license/jniltinho/gatus)](LICENSE)

Health dashboard that monitors HTTP, ICMP, TCP, DNS and other services, evaluates conditions on the status, response
time, body and certificates, sends alerts and shows the history of every check.

![Dashboard](docs/screenshots/dashboard.png)

## What it adds to the original Gatus

This project started as a fork of [TwiN/gatus](https://github.com/TwiN/gatus) and is now developed on its own. A
configuration file of the original works unchanged; on top of it:

| | |
|---|---|
| **Endpoint administration through the web** | Create, edit, disable and remove endpoints at `/admin`, without restarting Gatus. [docs/admin-endpoints.md](docs/admin-endpoints.md) |
| **Public status pages** | Pages open without login at `/status/<slug>`, with featured endpoints and a details page for every endpoint, while the dashboard stays protected. A page can also ask for a username and a password of its own. [docs/status-pages.md](docs/status-pages.md) |
| **MySQL and MariaDB storage** | `storage.type: mysql` for MySQL 8.4+ and MariaDB 10.11+, besides SQLite and PostgreSQL. [docs/storage-mysql.md](docs/storage-mysql.md) |
| **Push monitoring compatible with the Uptime Kuma** | Scripts and services report their status at `/api/push/<token>?status=up&msg=OK&ping=`, with Push endpoints, global keys, push on active endpoints and a Pending status with retries. [docs/push-monitoring.md](docs/push-monitoring.md) |
| **Response time chart of the Uptime Kuma** | Periods Recent, 3h, 6h, 24h and 1w, with the average, the minimum and the maximum, and columns for the failures. [docs/status-pages.md](docs/status-pages.md#response-time-chart) |
| **TLS certificate expiration** | Days until the certificate expires, below the name of the endpoint on the dashboard and, with `show-certificate-expiration`, on the status pages. [docs/status-pages.md](docs/status-pages.md) |
| **Login screen for `security.basic`** | A login page with logout instead of the browser dialog, with sessions stored in the database and a limit of failed logins, while `curl -u` keeps working. [docs/admin-endpoints.md](docs/admin-endpoints.md#login-screen) |
| **Backup and restore of the administration** | A JSON file with the endpoints, status pages and push keys, optionally encrypted, restored with a preview of what changes. [docs/admin-endpoints.md](docs/admin-endpoints.md#backup-and-restore) |
| **Command line** | `gatus config validate` checks a configuration before a deploy, `gatus password hash` generates the hash of a password, `gatus version` shows the build, and `gatus healthcheck` is the `HEALTHCHECK` of the image, which has no shell. [docs/cli.md](docs/cli.md) |
| **Bulk management script** | `docs/manager-gatus.py` registers a CSV of hosts, renames a group in every endpoint, lists endpoints and status pages and exports the push tokens, through the administration API. [docs/admin-endpoints.md](docs/admin-endpoints.md#managing-endpoints-from-the-command-line) |
| **Interface of its own** | Dark mode by default, square style, thin scrollbar in the colours of the theme and the Inter font served by Gatus itself, without calling any external service. |

**[See every screen →](docs/screenshots/README.md)**

## Quick start

With Docker, using a fixed version (`latest` is never published):

```bash
mkdir -p config && curl -sL -o config/config.yaml https://raw.githubusercontent.com/jniltinho/gatus/master/config.yaml
docker run -d --name gatus -p 127.0.0.1:8080:8080 -v "$PWD/config:/config" jniltinho/gatus:v6.0.1
```

Open http://127.0.0.1:8080. `docker ps` shows the container as `healthy` once `/health` answers.

That keeps nothing across restarts. With Docker Compose, the history in SQLite and the administration enabled:

```yaml
services:
  gatus:
    image: jniltinho/gatus:v6.0.1
    restart: unless-stopped
    ports:
      - "127.0.0.1:8080:8080"   # publish it through a reverse proxy, not directly
    volumes:
      - ./config:/config:ro
      - gatus-data:/data
volumes:
  gatus-data:
```

```bash
docker run --rm -it jniltinho/gatus:v6.0.1 password hash             # asks for the password, prints the hash
docker run --rm -v "$PWD/config:/config:ro" jniltinho/gatus:v6.0.1 config validate   # before every deploy
docker compose up -d
```

with the `storage`, `security` and `admin` blocks of the next section in `config/config.yaml`. For MariaDB or MySQL,
start from [.examples/docker-compose-mariadb-storage](.examples/docker-compose-mariadb-storage).

Without Docker, download `gatus_<version>_linux_<amd64|arm64>.tar.gz` from the
[releases](https://github.com/jniltinho/gatus/releases) and run:

```bash
tar xzf gatus_6.0.1_linux_amd64.tar.gz
./gatus config validate --config config.yaml
./gatus --config config.yaml
```

## Minimal configuration

```yaml
endpoints:
  - name: website
    url: "https://example.org"
    interval: 1m
    conditions:
      - "[STATUS] == 200"
      - "[RESPONSE_TIME] < 500"
```

To keep the history across restarts, configure `storage` (`sqlite`, `postgres` or `mysql`):

```yaml
storage:
  type: sqlite
  path: /data/data.db
```

The administration and the login screen need `security` and `admin.enabled: true`. The password is a bcrypt hash in
base64, which `gatus password hash` generates (see [docs/cli.md](docs/cli.md#gatus-password-hash)):

```yaml
security:
  basic:
    username: admin
    password-bcrypt-base64: "JDJhJDEwJD..."
admin:
  enabled: true
```

## Upgrading from v5.36.0-fork.N

Change the image tag or the binary: the configuration file, the database and the API routes are the same, and a backup
made on `v5.36.0-fork.27` restores on `v6.0.0` (this is tested by `test/e2e/upgrade.sh`). What changed with the new HTTP
server is listed in the [release notes](https://github.com/jniltinho/gatus/releases/tag/v6.0.0); the one that can bite
is that **paths are now case-sensitive** (`/HEALTH` and `/API/v1/...` answer `404`).

## Documentation

| Topic | Where |
|-------|-------|
| Screens | [docs/screenshots/README.md](docs/screenshots/README.md) |
| Full configuration: endpoints, conditions, alerting, storage, security, UI, suites, deployment and FAQ | [docs/README.md](docs/README.md) |
| Endpoint administration through the web, login screen, backup and restore | [docs/admin-endpoints.md](docs/admin-endpoints.md) |
| Public status pages and response time chart | [docs/status-pages.md](docs/status-pages.md) |
| Command line: `serve`, `version`, `config validate`, `password hash` and `healthcheck` | [docs/cli.md](docs/cli.md) |
| MySQL and MariaDB storage | [docs/storage-mysql.md](docs/storage-mysql.md) |
| Push monitoring compatible with the Uptime Kuma | [docs/push-monitoring.md](docs/push-monitoring.md) |
| Bulk management of endpoints and push tokens (`manager-gatus.py`) | [docs/admin-endpoints.md](docs/admin-endpoints.md#managing-endpoints-from-the-command-line) |
| Docker Compose examples | [.examples](.examples) |

## Build from source

```bash
make frontend-install frontend-build   # only when changing the web interface
make build                             # binary in dist/gatus
make lint test                         # go vet, gofmt and the Go tests
```

`main.go` only calls the commands of `cmd/`; every other Go package lives in `internal/` and is documented with
`go doc` (`go doc ./internal/api`, for example). The end-to-end suites are in `test/e2e/`.

The Go module is named `gatus/v5` and does not depend on the original repository, so it cannot be installed with
`go install`.

## Releases

Releases use `vX.Y.Z` tags (the older ones are `v5.36.0-fork.<N>`), with `linux/amd64` and `linux/arm64` tarballs on
[GitHub](https://github.com/jniltinho/gatus/releases) and the `jniltinho/gatus:<tag>` image on
[Docker Hub](https://hub.docker.com/r/jniltinho/gatus). There is no `latest` tag on purpose: pin the version you run.

## License

[Apache License 2.0](LICENSE), like the original Gatus by [TwiN](https://github.com/TwiN).
