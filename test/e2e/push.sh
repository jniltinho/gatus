#!/usr/bin/env bash
# End-to-end tests of the push monitoring and of its administration screens, with agent-browser.
#
#   test/e2e/push.sh
#
# Starts the locally built Gatus (temporary SQLite, basic auth, administration enabled), creates a global push key, a
# push endpoint and an active endpoint that accepts push through the administration screens, sends pushes with curl in
# the format of the Uptime Kuma and saves screenshots in dist/prints/push/ (dist/ is in .gitignore). Requires
# agent-browser with Chrome installed.
set -euo pipefail

ROOT=$(git rev-parse --show-toplevel)
cd "$ROOT"
PORT=${E2E_PORT:-18092}
BASE="http://127.0.0.1:$PORT"
PRINTS="$ROOT/dist/prints/push"
WORK=$(mktemp -d)
USERNAME=admin
PASSWORD='e2e-senha'
# bcrypt (cost 10) of PASSWORD, in base64 with the URL alphabet (as Gatus decodes it)
PASSWORD_HASH='JDJhJDEwJHo1LnE5empYYkN5Vm1Vd1RmNXZPMS5SeWRCdlc3UlMxMXBHdmpwcDBUUTZiMXlIQ1R3RVRT'
YAML_KEY='e2e-yaml-global-key-0123456789'
YAML_ENDPOINT_TOKEN='e2e-health-token'

command -v agent-browser >/dev/null || { echo "agent-browser not found"; exit 1; }
mkdir -p "$PRINTS"

STORAGE_TYPE=${E2E_STORAGE_TYPE:-sqlite}
STORAGE_PATH=${E2E_STORAGE_PATH:-$WORK/gatus.db}

echo "==> Building"
make -s build

cat > "$WORK/config.yaml" <<CONFIG
web:
  address: 127.0.0.1
  port: $PORT
storage:
  type: $STORAGE_TYPE
  path: "$STORAGE_PATH"
security:
  basic:
    username: $USERNAME
    password-bcrypt-base64: "$PASSWORD_HASH"
admin:
  enabled: true
push:
  keys:
    - name: yaml-key
      token: $YAML_KEY
  endpoints:
    - key: core_health
      token: $YAML_ENDPOINT_TOKEN
endpoints:
  - name: health
    group: core
    url: $BASE/health
    interval: 5s
    conditions:
      - "[STATUS] == 200"
# Fork: without pushes, the heartbeat records a pending result before each failure
external-endpoints:
  - name: late
    group: jobs
    token: late-job-token-000000000000000000
    heartbeat:
      interval: 10s
      retries: 1
CONFIG

GATUS_CONFIG_PATH="$WORK/config.yaml" dist/gatus > "$WORK/gatus.log" 2>&1 &
GATUS_PID=$!
admin() { agent-browser --session e2e-push-admin "$@"; }
cleanup() {
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
  admin screenshot --full "$PRINTS/error.png" >/dev/null 2>&1 || true
  tail -20 "$WORK/gatus.log"
  exit 1
}
testid() {
  echo "[data-testid=\"$1\"]"
}
push() {
  curl -s "$@"
}
authenticated() {
  curl -s -u "$USERNAME:$PASSWORD" "$@"
}

step "Push to an endpoint of the configuration file, with its token and with the global key of the file"
push "$BASE/api/push/$YAML_ENDPOINT_TOKEN?status=up&msg=OK&ping=12" | grep -q '"ok":true' || fail "push with the token of the endpoint was not accepted"
push "$BASE/api/push/$YAML_KEY/core_health?status=down&msg=from-yaml-key" | grep -q '"ok":true' || fail "push with the global key of the file was not accepted"
[ "$(curl -s -o /dev/null -w '%{http_code}' "$BASE/api/push/wrong-token-000")" = 404 ] || fail "expected 404 for an unknown token"

step "Push keys tab: the key of the file is read-only and a key is created through the web"
admin set viewport 1280 900 >/dev/null
# Signs in through the login screen of security.basic
login_screen() {
  admin open "$BASE/login" >/dev/null
  admin wait "$(testid login-username)" >/dev/null || fail "the login screen did not open"
  admin fill "$(testid login-username)" "$USERNAME" >/dev/null
  admin fill "$(testid login-password)" "$PASSWORD" >/dev/null
  admin click "$(testid login-submit)" >/dev/null
  admin wait "$(testid logout-button)" >/dev/null || fail "the login did not work"
}
login_screen
admin open "$BASE/admin/push-keys" >/dev/null
admin wait "$(testid push-key-row-config-yaml-key)" >/dev/null || fail "the key of the file is not listed"
admin fill "$(testid push-key-name)" "akamai" >/dev/null
admin click "$(testid push-key-create)" >/dev/null
admin wait "$(testid push-key-token)" >/dev/null || fail "the created key was not shown"
GLOBAL_KEY=$(admin get value "$(testid push-key-token)")
[ ${#GLOBAL_KEY} -eq 32 ] || fail "the created key does not have 32 characters"
admin wait "$(testid push-key-row-admin-akamai)" >/dev/null || fail "the created key is not listed"
[ "$(admin eval "document.documentElement.scrollHeight <= window.innerHeight + 1" | tr -d '"')" = true ] || fail "the push keys page should fit the window, with the scroll inside its table"
admin screenshot "$PRINTS/01-push-keys.png" >/dev/null
authenticated "$BASE/api/v1/admin/push-keys" | grep -q "$GLOBAL_KEY" && fail "the list of keys exposed the token"

step "New push endpoint: type Push, generated token, push URL and no Test button"
admin open "$BASE/admin/endpoints/new" >/dev/null
admin wait "$(testid admin-field-type)" >/dev/null
admin click "$(testid admin-field-type) button" >/dev/null
admin click "$(testid admin-type-push)" >/dev/null
admin wait "$(testid admin-push-url)" >/dev/null || fail "the push URL was not shown"
admin fill "$(testid admin-field-name)" "backup" >/dev/null
admin click "$(testid admin-field-group-select) button" >/dev/null
admin click "$(testid admin-group-new)" >/dev/null
admin fill "$(testid admin-field-group)" "jobs" >/dev/null
admin fill "$(testid admin-field-heartbeat)" "10m" >/dev/null
PUSH_TOKEN=$(admin get value "$(testid admin-field-push-token)")
[ ${#PUSH_TOKEN} -eq 32 ] || fail "the push token was not generated"
admin get value "$(testid admin-push-url)" | grep -q "/api/push/$PUSH_TOKEN?status=up&msg=OK&ping=" || fail "unexpected push URL"
[ "$(admin eval "document.querySelectorAll('[data-testid=\"admin-test\"]').length")" = 0 ] || fail "the Test button was shown for a push endpoint"
admin click "$(testid admin-mode-yaml)" >/dev/null
admin get value "$(testid admin-yaml)" | grep -q "type: push" || fail "the YAML does not contain type: push"
admin get value "$(testid admin-yaml)" | grep -q "url:" && fail "the YAML of a push endpoint contains url"
admin click "$(testid admin-mode-form)" >/dev/null
admin wait "$(testid admin-push-url)" >/dev/null
admin screenshot --full "$PRINTS/02-push-endpoint-form.png" >/dev/null
admin click "$(testid admin-save)" >/dev/null
admin wait "$(testid admin-row-jobs_backup)" >/dev/null || fail "the push endpoint was not created"

step "Pushes to the new endpoint, with its token and with the created global key"
push "$BASE/api/push/$PUSH_TOKEN?status=up&msg=backup%20ok&ping=250" | grep -q '"ok":true' || fail "push with the token was not accepted"
push -X POST "$BASE/api/push/$GLOBAL_KEY/jobs_backup?status=down&msg=disk%20full" | grep -q '"ok":true' || fail "push with the global key was not accepted"
authenticated "$BASE/api/v1/endpoints/jobs_backup/statuses" | grep -q "disk full" || fail "the message of the push is not in the history"

step "Active endpoint with Accept push"
admin open "$BASE/admin/endpoints/new" >/dev/null
admin wait "$(testid admin-field-name)" >/dev/null
admin fill "$(testid admin-field-name)" "cdn" >/dev/null
admin fill "$(testid admin-field-url)" "$BASE/health" >/dev/null
admin fill "$(testid admin-field-interval)" "5s" >/dev/null
admin click "$(testid admin-field-accept-push)" >/dev/null
admin wait "$(testid admin-push-url)" >/dev/null || fail "the push URL of the active endpoint was not shown"
ACTIVE_TOKEN=$(admin get value "$(testid admin-field-push-token)")
[ ${#ACTIVE_TOKEN} -eq 32 ] || fail "the push token of the active endpoint was not generated"
admin screenshot --full "$PRINTS/03-active-endpoint-accept-push.png" >/dev/null
admin click "$(testid admin-save)" >/dev/null
admin wait "$(testid admin-accepts-push-_cdn)" >/dev/null || fail "the list does not mark the active endpoint that accepts push"
admin wait "$(testid admin-accepts-push-core_health)" >/dev/null || fail "the list does not mark the endpoint of the file that accepts push"
[ "$(admin eval "document.documentElement.scrollHeight <= window.innerHeight + 1" | tr -d '"')" = true ] || fail "the endpoints page should fit the window, with the scroll inside its table"
admin screenshot "$PRINTS/04-endpoints.png" >/dev/null
push "$BASE/api/push/$ACTIVE_TOKEN?status=up&msg=akamai" | grep -q '"ok":true' || fail "push to the active endpoint was not accepted"
push "$BASE/api/push/$GLOBAL_KEY/_cdn?status=up&msg=akamai-global" | grep -q '"ok":true' || fail "push with the global key to the active endpoint was not accepted"

step "Editing: the token is shown, the type cannot become active and the push URL is kept"
admin open "$BASE/admin/endpoints/jobs_backup/edit" >/dev/null
admin wait "$(testid admin-push-toggle)" >/dev/null || fail "the push block was not shown when editing"
[ "$(admin eval "document.querySelectorAll('[data-testid=\"admin-push-url\"]').length" | tr -d '"')" = 0 ] || fail "the push block should start collapsed when editing"
admin get text "$(testid admin-push-summary)" | grep -q "Token …${PUSH_TOKEN: -4}" || fail "the collapsed push block does not summarize the token"
admin click "$(testid admin-push-toggle)" >/dev/null
admin wait "$(testid admin-push-url)" >/dev/null || fail "the push URL was not shown when editing"
[ "$(admin eval "Math.abs(document.querySelector('[data-testid=\"admin-generate-push-token\"]').getBoundingClientRect().height - document.querySelector('[data-testid=\"admin-field-push-token\"]').getBoundingClientRect().height)" | tr -d '"')" = 0 ] || fail "the Generate token button does not have the height of the input"
[ "$(admin eval "Math.abs(document.querySelector('[data-testid=\"admin-copy-push-url\"]').getBoundingClientRect().height - document.querySelector('[data-testid=\"admin-push-url\"]').getBoundingClientRect().height)" | tr -d '"')" = 0 ] || fail "the Copy button does not have the height of the push URL"
[ "$(admin get value "$(testid admin-field-push-token)")" = "$PUSH_TOKEN" ] || fail "the token shown when editing is different"
admin click "$(testid admin-field-type) button" >/dev/null
[ "$(admin eval "document.querySelectorAll('[data-testid=\"admin-type-http\"]').length")" = 0 ] || fail "an active type was offered for a push endpoint"
admin screenshot --full "$PRINTS/05-push-endpoint-edit.png" >/dev/null

step "Push endpoint with the token of an Uptime Kuma monitor and the Recent checks table"
KUMA_TOKEN='keSDu7G855jvVat1xWiY2Gk4CkL1End5'
admin open "$BASE/admin/endpoints/new" >/dev/null
admin wait "$(testid admin-field-type)" >/dev/null
admin click "$(testid admin-field-type) button" >/dev/null
admin click "$(testid admin-type-push)" >/dev/null
admin wait "$(testid admin-field-push-token)" >/dev/null
admin fill "$(testid admin-field-name)" "kuma-backup" >/dev/null
admin fill "$(testid admin-field-push-token)" "$KUMA_TOKEN" >/dev/null
admin get value "$(testid admin-push-url)" | grep -q "/api/push/$KUMA_TOKEN?status=up&msg=OK&ping=" || fail "the push URL does not use the pasted token"
admin click "$(testid admin-save)" >/dev/null
admin wait "$(testid admin-row-_kuma-backup)" >/dev/null || fail "the push endpoint with the pasted token was not created"
for message in "status=up&msg=Backup OK" "status=down&msg=Falha no backup: disco cheio" "status=up&msg=Backup recuperado"; do
  push -G "$BASE/api/push/$KUMA_TOKEN" --data-urlencode "${message%%&*}" --data-urlencode "${message#*&}" | grep -q '"ok":true' || fail "push '$message' was not accepted"
  sleep 0.2
done
selector_count() {
  admin eval "document.querySelectorAll('[data-testid=\"$1\"]').length" 2>/dev/null | tr -d '"'
}
admin open "$BASE/endpoints/jobs_backup" >/dev/null
admin wait "$(testid response-time-trend)" >/dev/null || fail "the Response Time Trend chart is not shown"
admin wait "$(testid recent-checks-card)" >/dev/null
top_of() {
  admin eval "Math.round(document.querySelector('[data-testid=\"$1\"]').getBoundingClientRect().top + window.scrollY)" 2>/dev/null | tr -d '"'
}
# Same order as the monitor page of the Uptime Kuma: heartbeat bars, numbers, chart and table of checks
BARS_TOP=$(top_of recent-checks-card)
CHART_TOP=$(top_of response-time-trend)
TABLE_TOP=$(top_of checks-table-card)
[ "$BARS_TOP" -lt "$CHART_TOP" ] && [ "$CHART_TOP" -lt "$TABLE_TOP" ] || fail "expected the bars, the chart and the table of checks in this order ($BARS_TOP, $CHART_TOP, $TABLE_TOP)"
[ "$(selector_count recent-checks-table)" = 0 ] || fail "the checks table should start collapsed"
admin wait 2000 >/dev/null
admin screenshot --full "$PRINTS/06-kuma-order.png" >/dev/null
admin open "$BASE/endpoints/_kuma-backup" >/dev/null
admin wait "$(testid recent-checks-toggle)" >/dev/null || fail "the toggle of the checks table is not shown"
[ "$(selector_count recent-checks-table)" = 0 ] || fail "the checks table should start collapsed"
admin click "$(testid recent-checks-toggle)" >/dev/null
admin wait "$(testid recent-check-2)" >/dev/null || fail "the Recent checks table does not have the 3 pushes"
recent_message() {
  admin get text "$(testid "recent-check-$1") $(testid recent-check-message)"
}
[ "$(recent_message 0)" = "Backup recuperado" ] || fail "unexpected most recent check: $(recent_message 0)"
[ "$(recent_message 1)" = "Falha no backup: disco cheio" ] || fail "unexpected second check: $(recent_message 1)"
[ "$(recent_message 2)" = "Backup OK" ] || fail "unexpected third check: $(recent_message 2)"
admin get text "$(testid recent-check-1)" | grep -q "Down" || fail "the failure is not shown as Down"
admin get text "$(testid recent-check-1)" | grep -q "Push" || fail "the origin is not shown as Push"
admin scrollintoview "$(testid recent-checks-table)" >/dev/null
admin screenshot "$PRINTS/06-recent-checks.png" >/dev/null
admin set media dark >/dev/null
admin open "$BASE/endpoints/_kuma-backup" >/dev/null
admin wait "$(testid recent-check-2)" >/dev/null
admin scrollintoview "$(testid recent-checks-table)" >/dev/null
admin screenshot "$PRINTS/07-recent-checks-dark.png" >/dev/null
admin open "$BASE/admin/push-keys" >/dev/null
admin wait "$(testid push-keys-table)" >/dev/null
admin screenshot --full "$PRINTS/08-push-keys-dark.png" >/dev/null
admin set media light >/dev/null

step "Pending: status=pending in yellow, retries of the heartbeat and panel of numbers with Ping"
push "$BASE/api/push/$KUMA_TOKEN?status=down&msg=Queda&ping=90" | grep -q '"ok":true' || fail "the down push was not accepted"
sleep 1
push "$BASE/api/push/$KUMA_TOKEN?status=up&msg=Voltou&ping=40" | grep -q '"ok":true' || fail "the up push was not accepted"
push "$BASE/api/push/$KUMA_TOKEN?status=pending&msg=Backup%20em%20andamento" | grep -q '"ok":true' || fail "the pending push was not accepted"
authenticated "$BASE/api/v1/endpoints/_kuma-backup/statuses" | python3 -c '
import json, sys
status = json.load(sys.stdin)
last = status["results"][-1]
sys.exit(0 if last.get("pending") and not last["success"] and last.get("message") == "Backup em andamento" and status.get("push") is True else 1)
' || fail "the pending push is not a pending result of a push endpoint in the API"
# The heartbeat of jobs_late ran without pushes since the start: Pending, then failures, and a single UNHEALTHY event
for _ in $(seq 1 30); do
  authenticated "$BASE/api/v1/endpoints/jobs_late/statuses" | python3 -c '
import json, sys
status = json.load(sys.stdin)
results = status["results"]
sys.exit(0 if len(results) >= 2 and results[0].get("pending") and not results[1].get("pending") and not results[1]["success"] else 1)
' && break
  sleep 1
done
authenticated "$BASE/api/v1/endpoints/jobs_late/statuses" | python3 -c '
import json, sys
status = json.load(sys.stdin)
results, events = status["results"], [event["type"] for event in status["events"]]
ok = len(results) >= 2 and results[0].get("pending") and results[0].get("message", "").startswith("heartbeat: no update received within") and not results[0].get("errors")
ok = ok and not results[1].get("pending") and not results[1]["success"] and events == ["START", "UNHEALTHY"]
sys.exit(0 if ok else 1)
' || fail "expected Pending, then a failure with a single UNHEALTHY event for the heartbeat with retries"
admin open "$BASE/endpoints/_kuma-backup" >/dev/null
admin wait "$(testid details-summary)" >/dev/null || fail "the panel of numbers is not shown"
admin get text "$(testid details-summary)" | grep -q "Ping (Current)" || fail "the panel of a push endpoint should show Ping"
[ "$(top_of recent-checks-card)" -lt "$(top_of details-summary)" ] && [ "$(top_of details-summary)" -lt "$(top_of response-time-trend)" ] || fail "expected the panel of numbers between the bars and the chart"
[ "$(selector_count recent-checks-table)" = 0 ] && admin click "$(testid recent-checks-toggle)" >/dev/null
admin wait "$(testid recent-check-0)" >/dev/null
admin get text "$(testid recent-check-0)" | grep -q "Pending" || fail "the pending push is not shown as Pending"
[ "$(admin eval "document.querySelector('[data-testid=\"recent-check-0\"]').innerHTML.includes('yellow')" 2>/dev/null | tr -d '"')" = true ] || fail "the Pending badge is not yellow"
admin wait 2000 >/dev/null
admin screenshot --full "$PRINTS/09-pending-light.png" >/dev/null
admin set media dark >/dev/null
admin open "$BASE/endpoints/_kuma-backup" >/dev/null
admin wait "$(testid details-summary)" >/dev/null
admin wait 2000 >/dev/null
admin screenshot --full "$PRINTS/10-pending-dark.png" >/dev/null
admin open "$BASE/" >/dev/null
admin wait "$(testid dashboard-summary-pending)" >/dev/null || fail "the dashboard summary does not count the pending endpoint"
admin screenshot "$PRINTS/11-dashboard-pending-dark.png" >/dev/null
admin set media light >/dev/null

step "Real time: a pending push shows up on the details page without reloading it"
admin open "$BASE/endpoints/_kuma-backup" >/dev/null
admin wait "$(testid details-summary)" >/dev/null
[ "$(selector_count recent-checks-table)" = 0 ] && admin click "$(testid recent-checks-toggle)" >/dev/null
admin wait "$(testid recent-check-0)" >/dev/null
# The marker only survives if the page is not reloaded
admin eval "window.__e2eNoReload = true" >/dev/null
admin wait 1500 >/dev/null
push "$BASE/api/push/$KUMA_TOKEN?status=pending&msg=Realtime%20pending" | grep -q '"ok":true' || fail "the real-time pending push was not accepted"
shown=false
for _ in $(seq 1 10); do
  if admin get text "$(testid recent-check-0)" 2>/dev/null | grep -q "Realtime pending"; then
    shown=true
    break
  fi
  sleep 0.5
done
[ "$shown" = true ] || fail "the pending push did not show up on the details page within 5 seconds"
[ "$(admin eval "window.__e2eNoReload === true" 2>/dev/null | tr -d '"')" = true ] || fail "the details page was reloaded"
admin get text "$(testid recent-check-0)" | grep -q "Pending" || fail "the real-time push is not shown as Pending"
# Chart in the format of the Uptime Kuma: the Down push is a red column and the Pending pushes are yellow columns
chart_data() {
  admin eval "document.querySelector('[data-testid=\"response-time-chart\"]').dataset.$1" 2>/dev/null | tr -d '"'
}
for _ in $(seq 1 10); do
  [ "$(chart_data pendingColumns)" -ge 2 ] && break
  sleep 0.5
done
[ "$(chart_data period)" = recent ] || fail "the chart did not open in Recent"
[ "$(chart_data pendingColumns)" -ge 2 ] || fail "the pending pushes are not yellow columns of the chart"
[ "$(chart_data downColumns)" -ge 1 ] || fail "the down push is not a red column of the chart"
[ "$(chart_data linePoints)" -ge 1 ] || fail "the chart has no line point"
admin wait 2000 >/dev/null
admin screenshot --full "$PRINTS/12-realtime-pending.png" >/dev/null
admin screenshot "$PRINTS/12-realtime-pending-viewport.png" >/dev/null

step "Chart periods: 24h and 1w with the aggregates, remembered after a reload"
admin network requests --clear >/dev/null 2>&1 || true
admin eval "(() => { const select = document.querySelector('[data-testid=\"response-time-chart-period\"]'); select.value = '24h'; select.dispatchEvent(new Event('change')) })()" >/dev/null
admin wait "[data-testid=\"response-time-chart\"][data-period=\"24h\"]" >/dev/null || fail "the chart did not load the 24h period"
# The down, up and pending pushes share a minute, which is a yellow (mixed) column
[ "$(( $(chart_data downColumns) + $(chart_data pendingColumns) ))" -ge 1 ] || fail "the minute of the down and pending pushes is not a column"
admin eval "(() => { const select = document.querySelector('[data-testid=\"response-time-chart-period\"]'); select.value = '1w'; select.dispatchEvent(new Event('change')) })()" >/dev/null
admin wait "[data-testid=\"response-time-chart\"][data-period=\"1w\"]" >/dev/null || fail "the chart did not load the 1w period"
requests=$(admin network requests 2>/dev/null)
grep -q "/api/v1/endpoints/_kuma-backup/response-time-chart?period=24h" <<<"$requests" || fail "the 24h chart was not requested"
grep -q "/api/v1/endpoints/_kuma-backup/response-time-chart?period=1w" <<<"$requests" || fail "the 1w chart was not requested"
authenticated "$BASE/api/v1/endpoints/_kuma-backup/response-time-chart?period=1w" | python3 -c '
import json, sys
chart = json.load(sys.stdin)
buckets = chart["buckets"]
sys.exit(0 if chart["bucketSeconds"] == 3600 and buckets and sum(b["pending"] for b in buckets) >= 2 and sum(b["down"] for b in buckets) >= 1 else 1)
' || fail "the 1w chart does not have the pending and down pushes"
admin reload >/dev/null
admin wait "[data-testid=\"response-time-chart\"][data-period=\"1w\"]" >/dev/null || fail "the period of the chart was not remembered after a reload"
admin wait 2000 >/dev/null
admin screenshot "$PRINTS/13-chart-1w.png" >/dev/null
admin set media dark >/dev/null
admin eval "(() => { const select = document.querySelector('[data-testid=\"response-time-chart-period\"]'); select.value = 'recent'; select.dispatchEvent(new Event('change')) })()" >/dev/null
admin wait "[data-testid=\"response-time-chart\"][data-period=\"recent\"]" >/dev/null
admin wait 2000 >/dev/null
admin screenshot "$PRINTS/14-chart-recent-dark.png" >/dev/null
admin set media light >/dev/null

step "Revoking the created key"
admin open "$BASE/admin/push-keys" >/dev/null
admin wait "$(testid push-key-revoke-akamai)" >/dev/null
admin click "$(testid push-key-revoke-akamai)" >/dev/null
admin wait "$(testid confirm-accept)" >/dev/null
admin click "$(testid confirm-accept)" >/dev/null
admin wait --text "revoked" >/dev/null || fail "the key was not revoked"
push "$BASE/api/push/$GLOBAL_KEY/jobs_backup?status=up" | grep -q '"ok":false' || fail "push with the revoked key was accepted"

echo "==> OK: push monitoring end-to-end tests passed (screenshots in $PRINTS)"
