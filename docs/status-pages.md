# Public status pages

> Feature exclusive to the [jniltinho/gatus](https://github.com/jniltinho/gatus) fork. In the original Gatus, the
> request for public status pages was closed as *not planned* ([TwiN/gatus#1311](https://github.com/TwiN/gatus/issues/1311)).

A status page is a public page, open **without login** at `/status/<slug>`, that only shows the selected groups and
endpoints, like the *Status Pages* of Uptime Kuma. The dashboard, the administration and the status API stay protected
by `security.basic` or `security.oidc`.

Each endpoint is shown with its name, current status, bars of the latest checks and uptime over 24 hours, 7 days and
30 days, and its name opens a public details page with the response time chart, like the endpoint details page of the
dashboard.

## Configuration

```yaml
status-pages:
  enabled: true                          # defaults to true; false unpublishes every page
  trusted-proxies: ["172.30.0.1/32"]     # see "Behind a reverse proxy"
  rate-limit: 120                        # 404 responses per minute per IP; 0 disables it
  pages:
    - slug: services
      title: "Services"
      description: "External websites and APIs"
      groups: [sites, apis]
      featured: [sites_github]           # shown at the top of the page, with more details
    - slug: infrastructure
      title: "Infrastructure"
      groups: [dns]
      endpoints: [core_database]         # keys of individual endpoints (group_name)
      enabled: false                     # pages of the file are published by default
```

| Field | Rule |
|-------|------|
| `slug` | 1 to 64 characters: lowercase letters, digits and hyphens, not starting or ending with a hyphen. Unique. `new`, `options`, `preview`, `validate` and `exposure` are reserved. |
| `title` | Required, up to 100 characters. |
| `description` | Optional, up to 1000 characters, plain text (no markdown or HTML). |
| `groups` | Up to 50 groups. Every enabled endpoint of the group is shown, including the ones created later. |
| `endpoints` | Up to 200 keys in the `group_name` format (the same key as the badges). |
| `featured` | Up to 10 endpoint keys shown at the top of the page, in cards with more details. They are part of the selection of the page and are not repeated in their group. |
| `charts` | **Deprecated and ignored.** Every endpoint of the page now has a details page with its response time chart. Still accepted, with a warning, so that pages saved by `v5.36.0-fork.2` stay valid; saving the page in the administration removes it. |
| `enabled` | Pages of the file: defaults to `true`. Pages managed through the web: defaults to `false`. |

A page must select at least one group, endpoint or featured endpoint. A group or key that does not exist yet does not invalidate the page:
the load logs a warning and the administration shows the warning when validating.

The default `config.yaml` of the fork (Docker image and release tarballs) already ships the `/status/services` and
`/status/infrastructure` pages with example endpoints.

## Pages managed through the web

With the [administration](admin-endpoints.md) enabled, the **Status pages** tab at `/admin/status-pages` lets you create,
edit, preview, publish, unpublish and remove pages. They are stored in the `managed_status_pages` table of the same
database as the endpoints (SQLite, PostgreSQL, MySQL or MariaDB).

- A new page is created **disabled**, also through the API: check the preview and tick **Published**.
- The pages of the configuration file are shown in the list for reference only.
- If the configuration file starts using the slug of a page managed through the web, the file wins: the web page is
  marked as in conflict and is not published until the file stops using the slug.
- Turning `admin.enabled` off does **not** unpublish the pages managed through the web (only the administration goes
  away). To take every page down, use `status-pages.enabled: false`.
- The endpoint form warns on which status pages the endpoint will be shown, by group or by key.

## What the page shows

- **Featured** endpoints first, in cards with the uptime and the average response time over 24 hours, 7 days and
  30 days, the last response time, the check bars and a **View details** link.
- The name of every endpoint links to its details page (see below).
- Sections in the order of `groups`, then the groups only reached through `endpoints` (in alphabetical order) and, last,
  **Other services** with the endpoints without group. Within each section, endpoints are sorted by name.
- Endpoints of the file, external endpoints and endpoints managed through the web, as long as they are enabled. Suites
  and `remote` instances are left out.
- At most 200 endpoints per page; above that, the page says that it only shows the first ones.
- The latest 50 results of each endpoint (or `storage.maximum-number-of-results`, if lower).

Statuses:

| Status | Endpoint | Group and page |
|--------|----------|----------------|
| Operational / Up | last result succeeded | every endpoint with results is up |
| Partial outage | — | some endpoints are up and others are down |
| Major outage / Down | last result failed | every endpoint with results is down |
| No data | no result yet | no endpoint has results |

The uptime is shown as "—" when there was no check during the period. With SQLite, PostgreSQL and MySQL, the history older
than 48 hours is aggregated per day, so the edges of the 7 and 30 day periods are approximate, as in the badges.

## Endpoint details page

`/status/<slug>/endpoints/<key>` is open without login for every endpoint shown on a published page, and has the layout
of the endpoint details page of the dashboard (`/endpoints/<key>`):

- current status, average response time and response time range of the latest checks, and time of the last check;
- the bars of the latest checks;
- **Response Time Trend**: the same chart as the dashboard, with the 24 hours / 7 days / 30 days selector and the
  unhealthy periods marked;
- response time, uptime and health badges;
- the events (monitoring started, became healthy, was unhealthy for…), the latest 50.

A key of an endpoint that is not on the page, or of a page that is not published, shows "Page not found". The page
refreshes every 60 seconds and pauses while the tab is hidden.

## Public API

`GET /api/v1/status-pages/<slug>` responds without authentication:

```json
{
  "slug": "services", "title": "Services", "description": "External websites and APIs",
  "status": "degraded", "updatedAt": "2026-09-14T19:30:00Z", "truncated": false,
  "featured": [{
    "name": "github", "group": "sites", "status": "up",
    "uptime": {"24h": 1, "7d": 0.999, "30d": 0.998},
    "responseTime": {"24h": 180, "7d": 175, "30d": 190},
    "results": [{"timestamp": "2026-09-14T19:29:30Z", "success": true, "durationMs": 171}]
  }],
  "groups": [{
    "name": "apis", "status": "degraded",
    "endpoints": [{
      "name": "github-api", "status": "up",
      "uptime": {"24h": 0.9993, "7d": 0.998, "30d": null},
      "responseTime": {"24h": 123, "7d": 130, "30d": null},
      "results": [{"timestamp": "2026-09-14T19:29:00Z", "success": true, "durationMs": 123}]
    }]
  }]
}
```

`GET /api/v1/status-pages/<slug>/endpoints/<key>` responds, also without authentication, the details of an endpoint
shown on the page, with its latest events (type and time only):

```json
{
  "page": {"slug": "services", "title": "Services"},
  "name": "github-api", "group": "apis", "status": "up", "updatedAt": "2026-09-14T19:30:00Z",
  "uptime": {"24h": 0.9993, "7d": 0.998, "30d": null},
  "responseTime": {"24h": 123, "7d": 130, "30d": null},
  "results": [{"timestamp": "2026-09-14T19:29:00Z", "success": true, "durationMs": 123}],
  "events": [{"type": "START", "timestamp": "2026-09-14T10:00:00Z"}, {"type": "HEALTHY", "timestamp": "2026-09-14T10:00:00Z"}]
}
```

The chart and the badges of the details page use the routes by key of the original Gatus
(`/api/v1/endpoints/<key>/response-times/<duration>/history` and `.../badge.svg`), which are already public.

**Never published by these routes:** URL, hostname, IP, port, HTTP status, errors, conditions, certificate or domain
expiration, alerts and `extra-labels`; the page payload has no key and no event.

- A missing, disabled or conflicting page, an invalid slug, a key of an endpoint that is not on the page and any other
  path under `/api/v1/status-pages` all respond the same `404 {"error":"status page not found"}`, with the same headers.
- `503 {"error":"status page temporarily unavailable"}` when the database could not be read; the error only goes to
  the log.
- The HTML route `/status/<anything>` always responds 200 with the application, which queries the API and shows
  "Page not found" when needed.
- Headers: `X-Robots-Tag: noindex, nofollow`, `X-Content-Type-Options: nosniff`,
  `Referrer-Policy: strict-origin-when-cross-origin`, `Cache-Control: no-cache` (200) and `no-store` (404, 429 and 503).
- The page can be embedded in an iframe (for example, on a monitoring TV). The public API does not send CORS headers.

## Cache and cost

- Each page is assembled at most once every 30 seconds, however many visitors it has: concurrent requests share the
  same assembly. A change made through the administration takes effect on the next request.
- The assembly reads the database in one transaction with three queries, whatever the number of endpoints, and at most
  4 pages are assembled at the same time.
- The details of an endpoint are assembled at most once every 30 seconds per page and endpoint, and a key that is not
  on the page responds 404 without reading the database.
- If the database fails, the error is cached for 5 seconds so as not to overload it.
- The page in the browser refreshes every 60 seconds and pauses while the tab is hidden.

## Rate limit

- Only **404** responses count towards the limit (`rate-limit` per minute per IP; IPv6 aggregated by /64). A published
  page is never blocked: its cost is already limited by the cache.
- When the limit is exceeded: `429 {"error":"too many requests"}` with `Retry-After`.

## Behind a reverse proxy

Without configuration, the IP of the visitor is the IP of the connection. Behind nginx, every visitor would arrive with
the same IP and share the same limit. Set in `trusted-proxies` where the proxy connects to Gatus from: `X-Forwarded-For`
is only read from those connections, from right to left, and the first IP that is not a trusted proxy identifies the
visitor.

When a connection from an untrusted private or local IP brings `X-Forwarded-For`, Gatus logs a warning (once per load)
with the IP to add to `trusted-proxies`, and the status page list of the administration shows the same warning.

### Docker with nginx on the host

With the port published on `127.0.0.1` only, Gatus sees connections coming from the **gateway of the Docker network**,
not from `127.0.0.1`. Pin the subnet of the compose network so that the gateway has a known IP:

```yaml
services:
  gatus:
    image: jniltinho/gatus:v5.36.0-fork.7
    ports:
      - "127.0.0.1:8080:8080"
    volumes:
      - ./config:/config:ro
      - ./data:/data
    networks:
      - gatus

networks:
  gatus:
    ipam:
      config:
        - subnet: 172.30.0.0/24
```

```yaml
status-pages:
  trusted-proxies: ["172.30.0.1/32"]
```

The nginx vhost must send `X-Forwarded-For`:

```nginx
location / {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
}
```

Alternatives: `network_mode: host` with `web.address: 127.0.0.1`, or the binary directly on the host; in both cases,
use `trusted-proxies: ["127.0.0.1/32", "::1/128"]`.

## Security

- **The slug is not access control.** Anyone with the address can see the page; do not publish names of services that
  must not be seen by third parties.
- The names of the endpoints and groups become public. Since selecting a group automatically includes new endpoints,
  check the exposure warning in the endpoint form.
- The badges, uptimes and response times by key (`/api/v1/endpoints/<key>/...`) are already public in the original
  Gatus and stay that way.

## Administration API

Routes under `/api/v1/admin`, with the same requirements as the [endpoint administration](admin-endpoints.md#api)
(authentication, permission, CSRF protection, body of up to 256 KB in JSON or YAML):

| Method and route | Purpose |
|------------------|---------|
| `GET /status-pages` | Lists the pages, with `publicationEnabled`, `managedUnavailable` and `sharedRateLimitWarning` |
| `GET /status-pages/options` | Groups and endpoints that can be selected |
| `GET /status-pages/exposure?group=<g>&key=<k>` | Pages on which an endpoint would be shown |
| `POST /status-pages/validate` | Validates a definition and returns warnings (`?slug=` to validate a change) |
| `POST /status-pages` | Creates (201 with `ETag`) |
| `GET /status-pages/<slug>` | Gets (with `ETag`) |
| `PUT /status-pages/<slug>` | Changes (requires `If-Match`) |
| `POST /status-pages/<slug>/enable` and `/disable` | Publishes or unpublishes (requires `If-Match`) |
| `DELETE /status-pages/<slug>` | Removes (requires `If-Match`) |
| `GET /status-pages/<slug>/preview` | Public payload of any page, including disabled ones, without cache |

Errors: 400 (invalid definition, reserved or changed slug), 404, 409 (slug in use or page of the file), 412 (outdated
version), 428 (missing `If-Match`), 501 (storage without support) and 503 (startup or reload in progress).

## Multiple instances with the same PostgreSQL, MySQL or MariaDB

A page created through the web on one instance only shows up on the others after they reload their configuration or
restart. Behind a load balancer, visitors alternate between the page and "Page not found" until every instance
reloads.

## Going back to the original Gatus

The `status-pages` section and the `managed_status_pages` table are ignored by the original Gatus, and the pages stop
existing. Back up the database before switching versions.

## End-to-end tests

```bash
test/e2e/status-pages.sh
```

Starts Gatus with a temporary SQLite database, basic auth and the administration, goes through the public page without
credentials (light and dark modes, 390 px, page not found, simulated OIDC) and the administration screens, with
screenshots in `dist/prints/status-pages/`.
