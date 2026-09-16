# AGENTS.fork.md

Rules for agents in the **jniltinho/gatus** fork. They complement the upstream [AGENTS.md](AGENTS.md); in case of conflict, these rules prevail.

## Required skills

For any Go task, load `golang-how-to`, which selects the other `golang-*` skills in `.claude/skills/` (code style, naming, error handling, concurrency, database, testing, security, lint, among others). For releases, use `create-release`. To test the interface through the browser, use `agent-browser`.

## Fork commands

- `make lint` — `go vet ./...` and `gofmt` only on the Go files added or changed since `UPSTREAM_BASE` (do not reformat upstream code)
- `make fmt` — applies `gofmt` to those same files
- `make build` — static binary in `dist/gatus`
- `make release-cross VERSION=5.36.0-fork.1` — `linux/amd64` and `linux/arm64` tarballs in `dist/`
- `make docker-release VERSION=5.36.0-fork.1` — publishes `jniltinho/gatus:v5.36.0-fork.1` (amd64 and arm64) to Docker Hub from the local machine; never publishes `latest`
- `make test` and the other targets of `AGENTS.md` still apply

`UPSTREAM_BASE` (in the `Makefile`) is the upstream commit the fork is based on.

## Go module

The module path of the fork is `gatus/v5` (upstream: `github.com/TwiN/gatus/v5`). Internal imports always use `gatus/v5/...`; never reintroduce `github.com/TwiN/gatus/v5`. The fork cannot be installed with `go install`/`go get`: build the binary with `make build` or use the tarballs and the image of the releases.

## Dependencies

- After adding or updating a Go dependency, run `go mod tidy`.
- Do **not** run `go mod vendor`: `vendor/` is not versioned in this repository (the instruction in `AGENTS.md` does not apply here).

## CI and releases

- `.github/workflows/ci.yml`: `make lint`, `make build` and `go test ./... -race` (with `sudo`, because of the ICMP test).
- `.github/workflows/release.yml`: triggered by `v*-fork.*` tags; builds the tarballs and the GitHub Release. It does not publish images.
- Fork tags: `v<upstream-version>-fork.<N>`. Never create `vX.Y.Z` tags without the suffix.

## Endpoint administration

Change (archived): `openspec/changes/archive/2026-09-15-add-admin-endpoint-management/` (read `design.md` before touching these areas); specs in `openspec/specs/admin-*`, `config-hot-reload`, `ui-square-style`, `ci-release-pipeline` and `agent-skills`.

- Endpoints managed through the web are stored in the `managed_endpoints` table; `cfg.Endpoints` only contains the YAML and is not changed after the load.
- The watchdog controls each endpoint through a registry (context and `done` per endpoint). Never change an `*endpoint.Endpoint` that is running: create a new object and restart it through the registry.
- The Prometheus labels are the ones registered in the current cycle; do not recompute the list outside of `InitializePrometheusMetrics`.
- Managed endpoints go through strict validation: no environment variable expansion and no fields that use credentials or files of the server.
- Administration writes are serialized with each other and with startup and hot reload.
- Keep new code in new files whenever possible, to reduce conflicts with upstream.
- Renaming (change `openspec/changes/archive/2026-09-15-rename-managed-endpoint/`, documentation in `docs/admin-endpoints.md#renaming`):
  - a changed name or group renames the key with `RenameManagedEndpoint`, which moves the `endpoints` row (the history
    follows `endpoint_id`) and the references of the managed status pages in the same transaction;
  - the monitoring of the old key stops before the transaction, and the triggered alerts are restored by the old key
    before it;
  - `statuspage` takes part through `managedendpoint.KeyRenameParticipant`, registered in its `init`, because
    `managedendpoint` cannot import it;
  - lock order: lifecycle change, `managedendpoint.statesMutex`, `statuspage.mutex`, transaction. `statuspage` must
    never take `statesMutex`.

## Public status pages

Change (archived): `openspec/changes/archive/2026-09-15-add-public-status-pages/` (read `design.md` before touching these areas; documentation in `docs/status-pages.md`); specs in `openspec/specs/public-status-pages`, `status-page-*`.

- The public payload only uses the types of `statuspage/payload.go`: never serialize `endpoint.Status` or `endpoint.Result` on a public route (hostname, errors and conditions would leak). The sanitization test decodes the JSON with `DisallowUnknownFields`.
- The public routes (`/api/v1/status-pages/*` and `/status/*`) are in the unprotected block of `api/api.go`, before the static files and the security middleware. The catch-all of `/api/v1/status-pages` is always registered: a path reaching the middleware would respond 401 and open the login prompt of the browser.
- Administration writes store to the database and **only after the commit** publish the snapshot with a new revision; the cache and `singleflight` use `slug|revision|generation`, never the database version.
- The assembly is synchronous in the goroutine of the request and resolves the store reader on every assembly (a reload closes and replaces the store).
- The limiter is our own and has no goroutine (the Fiber `limiter` leaks a goroutine per reload) and only counts 404 responses.
- The endpoint details page (`/status/<slug>/endpoints/<key>`, API `GET /api/v1/status-pages/<slug>/endpoints/<key>`) checks that the key is among the endpoints shown by the published page **before** reading the store; its chart reuses the upstream `ResponseTimeChart` with the public routes by key. `charts` is deprecated and ignored, but must stay accepted by the strict decoding (definitions saved by `v5.36.0-fork.2`).
- In the frontend, routes with `meta.public` do not show the login screen nor fetch `/api/v1/config`. The Tailwind version of the project (3.1.8) has no 950 shade: use `dark:bg-*-900/30`.
- The interface texts are in English, like the rest of the Gatus UI.

## Push monitoring

Change (archived): `openspec/changes/archive/2026-09-15-add-push-monitoring/` (read `design.md` before touching these areas; documentation in `docs/push-monitoring.md`); spec in `openspec/specs/push-monitoring`.

- The push route (`api/push.go`: `/api/push/:token`, `/api/push/:token/:key` and the catch-alls) is in the unprotected block of `api/api.go`, like the status pages: a path reaching the security middleware would respond 401. Inputs and responses must stay identical to the Uptime Kuma (`status`, `msg`, `ping` with the `parseFloat` rules, `{"ok":...}` bodies, 404 for every rejection).
- `push.Resolver` resolves every push from atomic snapshots, never from the database: the YAML (`cfg.ExternalEndpoints` and `push.endpoints`), `managedendpoint` (push index published with the states) and `pushkey` (global keys, compared by SHA-256 hash, published only after the commit). Never log tokens or keys.
- Heartbeats of Push endpoints run in the watchdog registry by key (`StartExternalEndpoint`), counting from the last accepted push (`lastPush`), not from the last stored result. Results of checks, pushes and heartbeats of a key are serialized by `lockEndpointResults`; pushes to registered endpoints go through `watchdog.SubmitEndpointResult`.
- The message and the origin of a result are in the fork table `endpoint_result_messages` (`ON DELETE CASCADE`), not in `endpoint_results`. The public payloads of the status pages must never carry errors, and only the details payload of a page with `show-messages` carries `message` and `origin`.
- Pending (change `openspec/changes/refine-push-status/`): `Result.Pending` with `Success: false`, stored in the `pending` column of `endpoint_result_messages`. Pending results skip `HandleAlerting` and never create events; `InsertEndpointResult` (SQL and memory) compares the other results with the last HEALTHY/UNHEALTHY event, never with the last result. Retries (`heartbeat.retries`) are applied in `processExternalEndpointResult` under the lock of the key, with a counter next to `lastPushes`, forgotten by `ForgetExternalEndpoint` on rename, delete and reload.
- Public messages (`show-messages`): only the details payload has `message`/`origin`, built by `publicMessage` from `ResultSummary` (message, heartbeat prefix in old errors, `HTTP <status>`). Never publish other errors.
- Managed definitions: `type: push` decodes into `endpoint.ExternalEndpoint`; the `push` option of active definitions is removed before the strict decoding. `token` and `push.token` are masked, and the detail returns `pushToken`.

## MySQL and MariaDB storage

Change (archived): `openspec/changes/archive/2026-09-15-add-mysql-storage/` (documentation in `docs/storage-mysql.md`); spec in `openspec/specs/mysql-storage`.

- The queries of `storage/store/sql` keep the PostgreSQL placeholders (`$N`): `mysql_connector.go` translates them, emulates `INSERT ... RETURNING <column>` and aborts a transaction at its first failed statement, like PostgreSQL. Never write `?` in a query.
- What MySQL cannot run as written lives in `dialect_mysql.go` (`dialectQuery` for the upserts, early branches for the deletions of old rows). The schema is `specific_mysql.go`: foreign keys as table constraints (MySQL 8.4 ignores `REFERENCES` in a column), `VARCHAR(768)` keys, `MEDIUMTEXT`, `DATETIME(6)`, indexes inside `CREATE TABLE`.
- Connections use `READ COMMITTED`; `InsertEndpointResult` and `InsertSuiteResult` retry deadlocks and lock wait timeouts up to 3 attempts.
- Tests: `GATUS_TEST_MYSQL_URL` and `GATUS_TEST_MARIADB_URL` (a user allowed to create databases: each test uses a database of its own). Local containers: `gatus-test-mysql` (mysql:8.4.11, port 53306) and `gatus-test-mariadb` (mariadb:10.11.19, port 53307). `conformance_test.go` compares every database with SQLite.
- Do not run two `go test` processes of `storage/store/sql` at the same time with `GATUS_TEST_POSTGRES_URL`: the PostgreSQL database is shared and `Clear` removes the data of the other process.

## Login screen of security.basic

Change (archived): `openspec/changes/archive/2026-09-15-add-basic-login-page/` (read `design.md` before touching these areas; documentation in `docs/admin-endpoints.md#login-screen`); specs in `openspec/specs/basic-login-page` and `admin-access-control`.

- Only with `security.basic` without OIDC (`security.Config.UsesBasicLogin`). `security/basic_auth.go` replaces the Fiber `basicauth`: a login session (`gatus_session` cookie, `login_sessions` table with only the SHA-256 of the token) or `Authorization: Basic`, under the failure limiter of `security/limiter.go`.
- The authentication runs once per request and is kept in the locals: the middleware, `IsAuthenticated` (`/api/v1/config`) and `IsAdmin` reuse it, so a wrong password counts one failure and runs bcrypt once. Never check the password outside of `checkCredentials`, nor before `Blocked`.
- Sessions are read from the store on every request, without cache and without goroutine: the login and the lookup that finds an expired session delete it. The credential fingerprint (username and hash) invalidates the sessions when the credential changes.
- `security` cannot import `statuspage` (cycle through `config`): `api/auth.go` computes the client IP with `statuspage.ClientIP` and `status-pages.trusted-proxies` and stores it in `security.LocalsClientIP`, before `/api/v1/config`, the login routes and the protected router.
- `POST /api/v1/auth/login` and `/logout` are always registered in the unprotected block (404 without basic login), with the origin rules of `api/admin_middleware.go`. Never log passwords or tokens.
- Frontend: `views/LoginPage.vue` (`meta.login`, no dashboard header), redirect validated by `utils/redirect.js` (`npm run test:unit`), calls to the protected API with `PROTECTED_API_HEADERS` and `notifyUnauthorized()` on 401 (`utils/auth.js`). Without these headers, a 401 carries `WWW-Authenticate: Basic` and the browser opens its native dialog.

## OpenSpec

- Proposals in `openspec/changes/<change>/`; validate with `openspec validate <change> --strict`.
- When a task of `tasks.md` is done, tick its checkbox.

## End-to-end tests

- Use the `agent-browser` skill. With the standalone binary, set `AGENT_BROWSER_SKILLS_DIR` to the `skill-data` of the installed version before `agent-browser skills get core`.
- Screenshots go to `dist/prints/`. `dist/` is in `.gitignore`: **never** commit screenshots.
- Scripts: `test/e2e/admin.sh`, `test/e2e/status-pages.sh`, `test/e2e/push.sh`, `test/e2e/certificate.sh` (local HTTPS server with a self-signed certificate; needs `openssl` and `python3`) and `test/e2e/login.sh`. The scripts sign in through the login screen (`login_screen`), not with `set credentials`. Wait for a selector (`wait "[data-testid=...]"`) instead of a text when the previous screen has the same text (for example, the "New status page" button and the title of the form).

## Syncing with upstream

```bash
git fetch upstream --tags
git merge upstream/master
# New upstream code comes with the old module path
grep -rl --include='*.go' 'github.com/TwiN/gatus/v5' . | xargs -r sed -i 's#github.com/TwiN/gatus/v5#gatus/v5#g'
gofmt -w $(git diff --name-only -- '*.go')
```

- Import conflicts (almost every Go file differs from upstream only by the module path): resolve them keeping the upstream content and apply the replacement above; check with `grep -rn 'github.com/TwiN/gatus/v5' --include='*.go' .` (no results) and `go build ./...`.
- `go.mod`: keep `module gatus/v5`.
- `go.mod`: keep the `replace` block of the TwiN modules (`third_party/github.com/TwiN/`). When the upstream requires another version of one of them, update its copy as described in `third_party/README.md` before `go mod tidy`; never remove the `replace` to download the module again.

- Conflict in `web/static/`: accept either side and regenerate with `make frontend-install && make frontend-build`.
- Workflows removed by the fork (`benchmark`, `labeler`, `publish-*`, `regenerate-static-assets`, `test`, `test-ui`): keep them removed.
- `AGENTS.md`: accept the upstream version and keep the line pointing to this file.
- `README.md`: the fork keeps a short README (summary, quick start and links); the full documentation of Gatus, taken from the upstream README, lives in `docs/README.md`. On a conflict or upstream change in `README.md`, keep the fork README and apply the upstream changes to `docs/README.md`, with the relative links one level up (`../.github/assets/`, `../.examples/`) and without the fork section. The upstream `AGENTS.md` rule "add to README.md" (alerting providers, for example) means `docs/README.md` in the fork.
- Update `UPSTREAM_BASE` in the `Makefile` to the incorporated upstream commit and run `make lint test`.
- MySQL storage: run the tests of `storage/store/sql` with `GATUS_TEST_MYSQL_URL` and `GATUS_TEST_MARIADB_URL`, and review the new upstream queries that use `RETURNING` with more than one column, `ON CONFLICT`, `LIMIT` in a subquery, `REFERENCES` in a column definition, `CREATE INDEX IF NOT EXISTS` or MySQL reserved words (quote them, like `"condition"`).
- The next release uses the new upstream version as its base (`vX.Y.Z-fork.1`).

## Commits and PRs

Follow the rule of `AGENTS.md`: commits and PRs made by an agent state that they were made by an agent, with the name and version of the model. Use the `feat:`, `fix:`, `chore:`, `docs:`, `ci:` prefixes (the `create-release` skill uses these prefixes for the notes).
