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
admin set credentials "$USERNAME" "$PASSWORD" >/dev/null
admin open "$BASE/admin/push-keys" >/dev/null
admin wait "$(testid push-key-row-config-yaml-key)" >/dev/null || fail "the key of the file is not listed"
admin fill "$(testid push-key-name)" "akamai" >/dev/null
admin click "$(testid push-key-create)" >/dev/null
admin wait "$(testid push-key-token)" >/dev/null || fail "the created key was not shown"
GLOBAL_KEY=$(admin get value "$(testid push-key-token)")
[ ${#GLOBAL_KEY} -eq 32 ] || fail "the created key does not have 32 characters"
admin wait "$(testid push-key-row-admin-akamai)" >/dev/null || fail "the created key is not listed"
admin screenshot --full "$PRINTS/01-push-keys.png" >/dev/null
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
admin screenshot --full "$PRINTS/04-endpoints.png" >/dev/null
push "$BASE/api/push/$ACTIVE_TOKEN?status=up&msg=akamai" | grep -q '"ok":true' || fail "push to the active endpoint was not accepted"
push "$BASE/api/push/$GLOBAL_KEY/_cdn?status=up&msg=akamai-global" | grep -q '"ok":true' || fail "push with the global key to the active endpoint was not accepted"

step "Editing: the token is shown, the type cannot become active and the push URL is kept"
admin open "$BASE/admin/endpoints/jobs_backup/edit" >/dev/null
admin wait "$(testid admin-push-url)" >/dev/null || fail "the push URL was not shown when editing"
[ "$(admin get value "$(testid admin-field-push-token)")" = "$PUSH_TOKEN" ] || fail "the token shown when editing is different"
admin click "$(testid admin-field-type) button" >/dev/null
[ "$(admin eval "document.querySelectorAll('[data-testid=\"admin-type-http\"]').length")" = 0 ] || fail "an active type was offered for a push endpoint"
admin screenshot --full "$PRINTS/05-push-endpoint-edit.png" >/dev/null

step "Revoking the created key"
admin open "$BASE/admin/push-keys" >/dev/null
admin wait "$(testid push-key-revoke-akamai)" >/dev/null
admin click "$(testid push-key-revoke-akamai)" >/dev/null
admin wait "$(testid confirm-accept)" >/dev/null
admin click "$(testid confirm-accept)" >/dev/null
admin wait --text "revoked" >/dev/null || fail "the key was not revoked"
push "$BASE/api/push/$GLOBAL_KEY/jobs_backup?status=up" | grep -q '"ok":false' || fail "push with the revoked key was accepted"

echo "==> OK: push monitoring end-to-end tests passed (screenshots in $PRINTS)"
