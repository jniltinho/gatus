[![Gatus](.github/assets/logo-with-dark-text.png)](https://github.com/jniltinho/gatus)

# Gatus (jniltinho/gatus fork)

Health dashboard that monitors HTTP, ICMP, TCP, DNS and other services, evaluates conditions on the status, response
time, body and certificates, sends alerts and shows the history of every check.

This is a fork of [TwiN/gatus](https://github.com/TwiN/gatus) that adds:

- **Endpoint administration through the web**: create, edit, disable and remove endpoints at `/admin`, without
  restarting Gatus. [docs/admin-endpoints.md](docs/admin-endpoints.md)
- **Public status pages**: pages open without login at `/status/<slug>`, with featured endpoints and a details page for
  every endpoint, while the dashboard stays protected. [docs/status-pages.md](docs/status-pages.md)
- **MySQL and MariaDB storage**: `storage.type: mysql` for MySQL 8.4+ and MariaDB 10.11+, besides SQLite and
  PostgreSQL. [docs/storage-mysql.md](docs/storage-mysql.md)
- **Push monitoring compatible with the Uptime Kuma**: scripts and services report their status at
  `/api/push/<token>?status=up&msg=OK&ping=`, with Push endpoints, global keys and push on active endpoints.
  [docs/push-monitoring.md](docs/push-monitoring.md)
- **TLS certificate expiration**: days until the certificate expires, discreetly below the name of the endpoint on the
  dashboard and, with `show-certificate-expiration`, on the status pages. [docs/status-pages.md](docs/status-pages.md)

![Gatus dashboard](.github/assets/dashboard-dark.jpg)

## Quick start

With Docker, using a fixed version (the fork does not publish `latest`):

```bash
mkdir -p config && curl -sL -o config/config.yaml https://raw.githubusercontent.com/jniltinho/gatus/master/config.yaml
docker run -d --name gatus -p 127.0.0.1:8080:8080 -v "$PWD/config:/config" jniltinho/gatus:v5.36.0-fork.12
```

Open http://127.0.0.1:8080.

Without Docker, download `gatus_<version>_linux_<amd64|arm64>.tar.gz` from the
[releases](https://github.com/jniltinho/gatus/releases) and run:

```bash
tar xzf gatus_5.36.0-fork.12_linux_amd64.tar.gz
GATUS_CONFIG_PATH=config.yaml ./gatus
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

To keep the history across restarts, configure `storage` (`sqlite`, `postgres` or `mysql`). The administration also
requires `security` and `admin.enabled: true`.

## Documentation

| Topic | Where |
|-------|-------|
| Full configuration: endpoints, conditions, alerting, storage, security, UI, suites, deployment and FAQ | [docs/README.md](docs/README.md) |
| Endpoint administration through the web | [docs/admin-endpoints.md](docs/admin-endpoints.md) |
| Public status pages | [docs/status-pages.md](docs/status-pages.md) |
| MySQL and MariaDB storage | [docs/storage-mysql.md](docs/storage-mysql.md) |
| Push monitoring compatible with the Uptime Kuma | [docs/push-monitoring.md](docs/push-monitoring.md) |
| Docker Compose examples | [.examples](.examples) |

## Build from source

```bash
make frontend-install frontend-build   # only when changing the web interface
make build                             # binary in dist/gatus
```

The Go module is named `gatus/v5` and does not depend on the original repository, so the fork cannot be installed with
`go install`.

## Releases

Fork releases use `v<upstream-version>-fork.<N>` tags, with `linux/amd64` and `linux/arm64` tarballs on
[GitHub](https://github.com/jniltinho/gatus/releases) and the `jniltinho/gatus:<tag>` image on
[Docker Hub](https://hub.docker.com/r/jniltinho/gatus).

## License

[Apache License 2.0](LICENSE), like the original Gatus by [TwiN](https://github.com/TwiN).
