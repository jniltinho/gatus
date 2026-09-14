# Endpoint administration through the web

> Feature exclusive to the [jniltinho/gatus](https://github.com/jniltinho/gatus) fork. The original Gatus only accepts
> endpoints in the configuration file ([TwiN/gatus#1345](https://github.com/TwiN/gatus/issues/1345)).

With the administration enabled, endpoints can be created, edited, disabled and removed at `/admin`, without access to
the server and without restarting Gatus. The endpoints of the configuration file keep working as always and are shown
in the administration for reference only.

## Requirements

- `security.basic` or `security.oidc` configured. With OIDC, `admin.allowed-subjects` is required.
- `storage.type` set to `sqlite`, `postgres` or `mysql` ([MySQL and MariaDB](storage-mysql.md)): the endpoints managed
  through the web are stored in the `managed_endpoints` table.

## Configuration

```yaml
storage:
  type: sqlite
  path: /data/data.db

security:
  basic:
    username: admin
    # bcrypt hash of the password, base64 encoded (see below)
    password-bcrypt-base64: "JDJhJDEwJHRiMnRFakxWazZLdXBzRERQazB1TE8vckRLY05Yb1hSdnoxWU0yQ1FaYXZRSW1McmladDYu"

admin:
  enabled: true
  # Required with security.oidc: subjects (sub claim) allowed to administer, case-insensitive
  # allowed-subjects: ["ops@example.com"]
  # Origins accepted for changes; needed behind a proxy that exposes another port
  # allowed-origins: ["https://status.example.com:8443"]

# Without any endpoint in the file, Gatus starts normally when admin.enabled is true
endpoints: []
```

To generate `password-bcrypt-base64`:

```bash
htpasswd -bnBC 10 "" 'your-password' | tr -d ':\n' | sed 's/$2y/$2a/' | base64 -w0 | tr '+/' '-_'
```

Gatus decodes this value with the URL base64 alphabet; `tr '+/' '-_'` prevents failures with hashes that produce `+`
or `/` in standard base64.

With `security.basic`, the only basic user is the administrator. With `security.oidc`, only the subjects of
`admin.allowed-subjects` can administer; the other authenticated users keep seeing the dashboard.

## Usage

- **List (`/admin`):** search by name, group or URL; shows the source (Web or YAML), endpoints in conflict with the YAML
  and invalid endpoints; lets you enable, disable and remove the endpoints managed through the web.
- **Form (`/admin/endpoints/new` and `/admin/endpoints/<key>/edit`):** form mode (name, group, URL, method, interval,
  conditions, headers, alerts and enabled) and YAML mode, with the same keys as an item of `endpoints` in the
  configuration file. The name and the group cannot be changed after creation.
- **Validate** checks the definition without saving it. **Test** runs a single check, without storing the result or
  triggering alerts, and shows each condition. **Save** stores and applies it.
- **Remove** deletes the definition and the whole history of the endpoint. Triggered alerts are not resolved with the
  alerting providers.

## Behavior

- Creating, changing, enabling, disabling or removing an endpoint takes effect immediately, without restarting Gatus
  and without restarting the other endpoints. A check in progress finishes before the change and its result is
  discarded.
- The history of the endpoints managed through the web is kept across restarts and reloads of the configuration file.
- If the configuration file starts defining the same key (group and name) as an endpoint managed through the web, the
  one from the file wins and the one from the web is marked as in conflict, without being deleted. Removing the web one,
  in that case, does not affect the one from the file.
- Turning `admin.enabled` off only hides the administration: the endpoints managed through the web keep being monitored.
- During startup or a configuration reload, changes respond 503 and nothing is stored.
- With `skip-invalid-config-update: true`, an invalid configuration file no longer brings Gatus down: the previous
  configuration stays in use until the file is fixed.

## Restrictions of the endpoints managed through the web

- Environment variables (`$VAR`, `${VAR}`) are not expanded.
- `client.identity-aware-proxy`, `client.tls.certificate-file`, `client.tls.private-key-file`, `store` and `always-run`,
  which would use credentials or files of the server, are not accepted.
- `client.tunnel` must exist in `tunneling`.
- Alerts need a provider configured in `alerting`; invalid overrides are rejected.
- `extra-labels` can only use labels that already exist in the endpoints of the configuration file (Prometheus metrics).
- The key cannot be the same as the key of an endpoint, external endpoint, suite or suite endpoint of the file.

## Secrets

Every response masks with `********`: headers whose name contains `authorization`, `cookie`, `token`, `secret`,
`password` or `key`; the password and sensitive parameters of the URL; `client.oauth2.client-secret`; `ssh.password`;
`ssh.private-key`; and the values of `alerts[].provider-override`. When saving, a value equal to the mask keeps the
stored value. Secrets are stored as plain text in the database, as they would be in the configuration file: protect
your backups.

## API

Every route requires administrator authentication. Changes require `Content-Type` `application/json` or
`application/yaml` when there is a body, and the `Origin` of the browser must match the address of Gatus.

| Method and route | Description |
|------------------|-------------|
| `GET /api/v1/admin/endpoints` | Lists the endpoints of the file and the ones managed through the web |
| `GET /api/v1/admin/endpoints/{key}` | Stored and effective definition (with `ETag`) |
| `POST /api/v1/admin/endpoints` | Creates (201) |
| `PUT /api/v1/admin/endpoints/{key}` | Changes (requires `If-Match`) |
| `POST /api/v1/admin/endpoints/{key}/enable` and `/disable` | Enables or disables (requires `If-Match`) |
| `DELETE /api/v1/admin/endpoints/{key}` | Removes (requires `If-Match`) |
| `POST /api/v1/admin/endpoints/validate[?key=]` | Validates without saving |
| `POST /api/v1/admin/endpoints/test[?key=]` | Tests once (at most 2 concurrent tests) |
| `POST /api/v1/admin/endpoints/parse` | Converts YAML into a JSON document, without validating |
| `GET /api/v1/admin/metadata` | Available alert types, tunnels and labels |

Status codes: 400 invalid definition or changed name/group; 404 not found; 409 key in use or endpoint of the
configuration file; 412 outdated version; 413 body above 256 KB; 415 invalid content type; 428 missing `If-Match`;
429 too many tests; 503 startup or reload in progress.

```bash
curl -u admin:your-password -H 'Content-Type: application/yaml' \
  --data-binary $'name: site\ngroup: web\nurl: https://example.com\nconditions: ["[STATUS] == 200"]\n' \
  http://127.0.0.1:8080/api/v1/admin/endpoints
```

## Behind a reverse proxy

The origin expected for changes is derived from the `Host` header and the scheme (`X-Forwarded-Proto`). With nginx,
forward these headers:

```nginx
proxy_set_header Host $host;
proxy_set_header X-Forwarded-Proto $scheme;
```

If the public address uses a port different from the one forwarded in `Host`, list it in `admin.allowed-origins`.

## Multiple instances with the same PostgreSQL, MySQL or MariaDB

A change made on one instance only takes effect on the others after they restart or reload their configuration. In the
meantime, an instance that still monitors a removed endpoint may recreate its history.

## Versions and images

- Fork releases use `v<upstream-version>-fork.<N>` tags (e.g. `v5.36.0-fork.1`). Because of SemVer precedence, these
  tags sort below the upstream version with the same base in tools such as Renovate and `sort -V`.
- Docker Hub image: `jniltinho/gatus:<tag>` (`linux/amd64` and `linux/arm64`), published with
  `make docker-release VERSION=<version without the v>`. The `latest` tag is not published.

## Going back to the original Gatus

The original image ignores the `managed_endpoints` table: the endpoints managed through the web stop being monitored
and their history is deleted on the first start. Before going back, back up the database and make sure the
configuration file has at least one endpoint, otherwise the original image does not start.

## End-to-end tests

`test/e2e/admin.sh` starts a local Gatus with a temporary SQLite database and goes through the screens with
[agent-browser](https://github.com/vercel-labs/agent-browser), saving screenshots in `dist/prints/` (outside of git).
