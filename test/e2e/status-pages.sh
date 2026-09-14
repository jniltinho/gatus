#!/usr/bin/env bash
# End-to-end tests of the public status pages and of their administration screens, with agent-browser.
#
#   test/e2e/status-pages.sh
#
# Starts the locally built Gatus (temporary SQLite, basic auth, administration enabled, local endpoints), opens the
# public page in a session without credentials and the administration screens in a session with credentials, and saves
# screenshots in dist/prints/status-pages/ (dist/ is in .gitignore). Requires agent-browser with Chrome installed.
set -euo pipefail

ROOT=$(git rev-parse --show-toplevel)
cd "$ROOT"
PORT=${E2E_PORT:-18091}
BASE="http://127.0.0.1:$PORT"
PRINTS="$ROOT/dist/prints/status-pages"
WORK=$(mktemp -d)
USERNAME=admin
PASSWORD='e2e-senha'
# bcrypt (cost 10) of PASSWORD, in base64 with the URL alphabet (as Gatus decodes it)
PASSWORD_HASH='JDJhJDEwJHo1LnE5empYYkN5Vm1Vd1RmNXZPMS5SeWRCdlc3UlMxMXBHdmpwcDBUUTZiMXlIQ1R3RVRT'

command -v agent-browser >/dev/null || { echo "agent-browser not found"; exit 1; }
mkdir -p "$PRINTS"

echo "==> Building"
make -s build

cat > "$WORK/config.yaml" <<CONFIG
web:
  address: 127.0.0.1
  port: $PORT
storage:
  type: sqlite
  path: $WORK/gatus.db
security:
  basic:
    username: $USERNAME
    password-bcrypt-base64: "$PASSWORD_HASH"
admin:
  enabled: true
endpoints:
  - name: health
    group: core
    url: $BASE/health
    interval: 5s
    conditions:
      - "[STATUS] == 200"
  - name: offline
    group: core
    url: http://127.0.0.1:1/
    interval: 5s
    conditions:
      - "[STATUS] == 200"
  - name: panel
    url: $BASE/health
    interval: 5s
    conditions:
      - "[STATUS] == 200"
status-pages:
  rate-limit: 0
  pages:
    - slug: services
      title: "Services"
      description: "End-to-end test page"
      groups: [core]
      endpoints: [_panel]
    - slug: draft
      title: "Draft"
      groups: [core]
      enabled: false
CONFIG

GATUS_CONFIG_PATH="$WORK/config.yaml" dist/gatus > "$WORK/gatus.log" 2>&1 &
GATUS_PID=$!
public() { agent-browser --session e2e-status-public "$@"; }
admin() { agent-browser --session e2e-status-admin "$@"; }
cleanup() {
  public close >/dev/null 2>&1 || true
  admin close >/dev/null 2>&1 || true
  kill "$GATUS_PID" >/dev/null 2>&1 || true
  wait "$GATUS_PID" 2>/dev/null || true
  rm -rf "$WORK"
}
trap cleanup EXIT

for _ in $(seq 1 60); do
  curl -sf "$BASE/health" >/dev/null && break
  sleep 1
done
curl -sf "$BASE/health" >/dev/null || { echo "Gatus did not start"; cat "$WORK/gatus.log"; exit 1; }

STEP=0
step() {
  STEP=$((STEP + 1))
  echo "==> $STEP. $*"
}
fail() {
  echo "FAILED: $*"
  public screenshot --full "$PRINTS/error-public.png" >/dev/null 2>&1 || true
  admin screenshot --full "$PRINTS/error-admin.png" >/dev/null 2>&1 || true
  tail -20 "$WORK/gatus.log"
  exit 1
}
testid() {
  echo "[data-testid=\"$1\"]"
}
api_status() {
  curl -s -o /dev/null -w '%{http_code}' "$@"
}
body_text() {
  "$1" eval "document.body.innerText" 2>/dev/null
}
js() {
  local session=$1
  shift
  "$session" eval "$*" 2>/dev/null | tr -d '"'
}

step "Public API without credentials and protected routes"
[ "$(api_status "$BASE/api/v1/status-pages/services")" = 200 ] || fail "expected 200 from the public API"
[ "$(api_status "$BASE/api/v1/status-pages/draft")" = 404 ] || fail "expected 404 for the disabled page"
[ "$(api_status "$BASE/api/v1/status-pages/a/b")" = 404 ] || fail "expected 404 for an invalid path, without 401"
[ "$(api_status "$BASE/status/missing")" = 200 ] || fail "expected 200 from the HTML route"
[ "$(api_status "$BASE/api/v1/endpoints/statuses")" = 401 ] || fail "expected 401 from the protected routes"
if curl -s "$BASE/api/v1/status-pages/services" | grep -qE '127\.0\.0\.1|connection refused|core_health'; then
  fail "the public API exposed a URL, an error or a key"
fi

step "Public page in light mode, without credentials"
public set viewport 1280 900 >/dev/null
public set media light >/dev/null
public open "$BASE/status/services" >/dev/null
public wait --text "Partial outage" >/dev/null || fail "the page did not show the partial outage"
public wait 700 >/dev/null
public screenshot --full "$PRINTS/01-services-light.png" >/dev/null
[ "$(js public 'document.title')" = "Services" ] || fail "document.title is not the title of the page"
[ "$(js public 'document.documentElement.lang')" = "en" ] || fail "the public layout must not change the language of the document"
grep -q "Other services" <<<"$(body_text public)" || fail "the endpoint without group is not in the Other services section"
requests=$(public network requests 2>/dev/null)
grep -q "/api/v1/config" <<<"$requests" && fail "the public page called /api/v1/config"
grep -qE '\b401\b' <<<"$requests" && fail "a request of the public page received 401"

step "Detail of a check with the keyboard"
public eval "document.querySelector('[data-testid=\"status-endpoint-health\"] [role=group]').focus()" >/dev/null
public press ArrowLeft >/dev/null
public wait 300 >/dev/null
grep -q " ms" <<<"$(js public "document.querySelector('[data-testid=\"status-endpoint-health\"] [data-testid=status-endpoint-detail]').textContent")" || fail "the keyboard did not show the detail of the check"
public screenshot "$PRINTS/02-services-keyboard-detail.png" >/dev/null

step "Dark mode and 390 px screen"
public set media dark >/dev/null
public open "$BASE/status/services" >/dev/null
public wait "$(testid status-summary)" >/dev/null || fail "the status banner did not show up"
public wait 700 >/dev/null
public screenshot --full "$PRINTS/03-services-dark.png" >/dev/null
public set viewport 390 844 >/dev/null
public open "$BASE/status/services" >/dev/null
public wait "$(testid status-summary)" >/dev/null || fail "the status banner did not show up at 390 px"
public wait 700 >/dev/null
public screenshot --full "$PRINTS/04-services-390px-dark.png" >/dev/null
[ "$(js public "document.querySelectorAll('[data-testid=\"status-endpoint-health\"] [role=group] > span').length")" = 25 ] || fail "expected 25 bars on a narrow screen"
public set viewport 1280 900 >/dev/null
public set media light >/dev/null

step "Missing page, disabled page and malformed slug"
public network requests --clear >/dev/null 2>&1 || true
for path in missing draft "a%2Fb" "a/b"; do
  public open "$BASE/status/$path" >/dev/null
  public wait --text "Page not found" >/dev/null || fail "/status/$path did not show Page not found"
done
public screenshot "$PRINTS/05-page-not-found.png" >/dev/null
requests=$(public network requests 2>/dev/null)
grep -q "/api/v1/status-pages/a" <<<"$requests" && fail "the malformed slug called the API"
grep -qE '\b401\b' <<<"$requests" && fail "a page not found received 401"

step "Simulated OIDC: no login screen on the public page; positive control on the dashboard"
public network route "**/api/v1/config" --body '{"oidc":true,"authenticated":false}' >/dev/null
public open "$BASE/status/services" >/dev/null
public wait "$(testid status-page-title)" >/dev/null || fail "the public page did not load with simulated OIDC"
grep -q "Login with OIDC" <<<"$(body_text public)" && fail "the public page showed the login screen"
public open "$BASE/" >/dev/null
public wait --text "Login with OIDC" >/dev/null || fail "the positive control did not show the login screen"
public screenshot "$PRINTS/06-oidc-dashboard-control.png" >/dev/null
public network unroute >/dev/null 2>&1 || true

step "Administration: list with the pages of the configuration file"
admin set credentials "$USERNAME" "$PASSWORD" >/dev/null
admin set viewport 1280 900 >/dev/null
admin set media light >/dev/null
admin open "$BASE/admin/status-pages" >/dev/null
admin wait "$(testid status-page-row-config-services)" >/dev/null || fail "the list did not show the services page"
admin wait "$(testid status-page-row-config-draft)" >/dev/null || fail "the list did not show the draft page"
admin screenshot --full "$PRINTS/07-admin-list.png" >/dev/null

step "Administration: create with the form and validate"
admin click "$(testid admin-new-status-page)" >/dev/null
admin wait "$(testid status-page-field-slug)" >/dev/null || fail "the form did not open"
admin fill "$(testid status-page-field-slug)" "team" >/dev/null
admin fill "$(testid status-page-field-title)" "Team" >/dev/null
admin fill "$(testid status-page-field-description)" "Services used by the team" >/dev/null
admin click "$(testid status-page-group-core)" >/dev/null
admin fill "$(testid status-page-endpoint-search)" "panel" >/dev/null
admin click "$(testid status-page-endpoint-_panel)" >/dev/null
admin click "$(testid status-page-validate)" >/dev/null
admin wait --text "The page will show 3 endpoints" >/dev/null || fail "the validation did not count the 3 endpoints"
admin screenshot --full "$PRINTS/08-admin-validation.png" >/dev/null

step "Administration: save (created disabled) and preview"
admin click "$(testid status-page-save)" >/dev/null
admin wait --text "Status page created" >/dev/null || fail "the creation was not confirmed"
[ "$(js admin 'location.pathname')" = "/admin/status-pages/team/edit" ] || fail "the creation did not lead to the edition"
[ "$(api_status "$BASE/api/v1/status-pages/team")" = 404 ] || fail "the page that was just created should not be public"
admin click "$(testid status-page-preview-button)" >/dev/null
admin wait "$(testid status-page-preview)" >/dev/null || fail "the preview did not show up"
admin screenshot --full "$PRINTS/09-admin-preview.png" >/dev/null

step "Administration: publish and open without credentials"
admin click "$(testid status-page-field-enabled)" >/dev/null
admin click "$(testid status-page-save)" >/dev/null
admin wait --text "saved and published" >/dev/null || fail "the publication was not confirmed"
public open "$BASE/status/team" >/dev/null
public wait --text "Services used by the team" >/dev/null || fail "the published page did not open without credentials"
public screenshot --full "$PRINTS/10-team-public.png" >/dev/null

step "Administration: exposure warning in the endpoint form"
admin open "$BASE/admin/endpoints/new" >/dev/null
admin wait "$(testid admin-field-group)" >/dev/null || fail "the endpoint form did not open"
admin fill "$(testid admin-field-group)" "core" >/dev/null
admin fill "$(testid admin-field-name)" "new" >/dev/null
admin wait "$(testid admin-endpoint-exposure)" >/dev/null || fail "the exposure warning did not show up"
exposure=$(js admin "document.querySelector('[data-testid=admin-endpoint-exposure]').innerText")
grep -q "Team" <<<"$exposure" && grep -q "by group" <<<"$exposure" || fail "the exposure warning did not mention the Team page by group"
admin screenshot "$PRINTS/11-admin-exposure.png" >/dev/null

step "Administration: dark mode and removal"
admin set media dark >/dev/null
admin open "$BASE/admin/status-pages" >/dev/null
admin wait "$(testid status-page-row-admin-team)" >/dev/null || fail "the list did not show the team page"
grep -q "Published" <<<"$(js admin "document.querySelector('[data-testid=\"status-page-row-admin-team\"]').innerText")" || fail "the team page is not shown as published"
admin wait 700 >/dev/null
admin screenshot --full "$PRINTS/12-admin-list-dark.png" >/dev/null
admin click "$(testid status-page-remove-team)" >/dev/null
admin click "$(testid confirm-accept)" >/dev/null
admin wait --text "removed" >/dev/null || fail "the removal was not confirmed"
[ "$(api_status "$BASE/api/v1/status-pages/team")" = 404 ] || fail "the removed page is still public"

echo "OK: $STEP steps; screenshots in $PRINTS"
