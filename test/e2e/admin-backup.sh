#!/usr/bin/env bash
# End-to-end tests of the backup and restore of the administration, with agent-browser.
#
#   test/e2e/admin-backup.sh
#
# Starts a source installation (temporary SQLite), registers an endpoint, a status page and a push key through the web,
# downloads the backup with and without password, then starts a target installation, whose configuration file has a
# status page with the same slug, and restores the backup through the Backup tab. Screenshots in
# dist/prints/admin-backup/ (dist/ is in .gitignore). Requires agent-browser with Chrome installed.
set -euo pipefail

ROOT=$(git rev-parse --show-toplevel)
cd "$ROOT"
PORT=${E2E_PORT:-18093}
BASE="http://127.0.0.1:$PORT"
PRINTS="$ROOT/dist/prints/admin-backup"
WORK=$(mktemp -d)
USERNAME=admin
PASSWORD='e2e-senha'
# bcrypt (cost 10) of PASSWORD, in base64 with the URL alphabet (as Gatus decodes it)
PASSWORD_HASH='JDJhJDEwJHo1LnE5empYYkN5Vm1Vd1RmNXZPMS5SeWRCdlc3UlMxMXBHdmpwcDBUUTZiMXlIQ1R3RVRT'
BACKUP_PASSWORD='correct horse battery'
PUSH_TOKEN='keSDu7G855jvVat1xWiY2Gk4CkL1End5'

command -v agent-browser >/dev/null || { echo "agent-browser not found"; exit 1; }
mkdir -p "$PRINTS"

echo "==> Building"
make -s build

write_config() {
  local name=$1 extra=$2
  cat > "$WORK/$name.yaml" <<CONFIG
web:
  address: 127.0.0.1
  port: $PORT
storage:
  type: sqlite
  path: "$WORK/$name.db"
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
    interval: 30s
    conditions:
      - "[STATUS] == 200"
$extra
CONFIG
}
write_config source ""
write_config target "status-pages:
  pages:
    - slug: jobs
      title: Jobs of the file
      groups: [jobs]"

GATUS_PID=""
start_gatus() {
  GATUS_CONFIG_PATH="$WORK/$1.yaml" dist/gatus > "$WORK/$1.log" 2>&1 &
  GATUS_PID=$!
  for _ in $(seq 1 60); do
    curl -sf "$BASE/health" >/dev/null && return 0
    sleep 1
  done
  echo "Gatus did not start"; cat "$WORK/$1.log"; exit 1
}
stop_gatus() {
  if [ -n "$GATUS_PID" ]; then
    kill "$GATUS_PID" >/dev/null 2>&1 || true
    wait "$GATUS_PID" 2>/dev/null || true
    GATUS_PID=""
  fi
}
admin() { agent-browser --session e2e-admin-backup "$@"; }
cleanup() {
  admin close >/dev/null 2>&1 || true
  stop_gatus
  rm -rf "$WORK"
}
trap cleanup EXIT

STEP=0
step() {
  STEP=$((STEP + 1))
  echo "==> $STEP. $*"
}
fail() {
  echo "FAILED: $*"
  admin screenshot --full "$PRINTS/error.png" >/dev/null 2>&1 || true
  tail -20 "$WORK"/*.log
  exit 1
}
testid() {
  echo "[data-testid=\"$1\"]"
}
js() {
  admin eval "$*" 2>/dev/null | tr -d '"'
}
authenticated() {
  curl -s -u "$USERNAME:$PASSWORD" "$@"
}
login_screen() {
  admin open "$BASE/login" >/dev/null
  admin wait "$(testid login-username)" >/dev/null || fail "the login screen did not open"
  admin fill "$(testid login-username)" "$USERNAME" >/dev/null
  admin fill "$(testid login-password)" "$PASSWORD" >/dev/null
  admin click "$(testid login-submit)" >/dev/null
  admin wait "$(testid logout-button)" >/dev/null || fail "the login did not work"
}
plan_action() {
  js "document.querySelector('[data-testid=\"restore-plan-row-$1-$2\"]')?.dataset.action || ''"
}
result_of() {
  js "document.querySelector('[data-testid=\"restore-result-row-$1-$2\"]')?.dataset.result || ''"
}

step "Source installation: an endpoint, a status page and a push key registered through the web"
start_gatus source
authenticated -H 'Content-Type: application/yaml' --data-binary "type: push
name: backup
group: jobs
token: $PUSH_TOKEN" "$BASE/api/v1/admin/endpoints" | grep -q '"key":"jobs_backup"' || fail "the push endpoint was not created"
authenticated -H 'Content-Type: application/yaml' --data-binary "slug: jobs
title: Jobs
groups: [jobs]
enabled: true" "$BASE/api/v1/admin/status-pages" | grep -q '"slug":"jobs"' || fail "the status page was not created"
KEY_TOKEN=$(authenticated -H 'Content-Type: application/json' -d '{"name":"akamai"}' "$BASE/api/v1/admin/push-keys" | python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])')
[ -n "$KEY_TOKEN" ] || fail "the push key was not created"

step "Backup tab: counts, download without and with password"
admin set viewport 1280 900 >/dev/null
login_screen
admin open "$BASE/admin/backup" >/dev/null
admin wait "$(testid admin-backup)" >/dev/null || fail "the Backup tab did not open"
admin wait 1000 >/dev/null
grep -q "1" <<<"$(js "document.querySelector('[data-testid=\"backup-section-download\"]').innerText")" || fail "the counts are not shown"
[ "$(js "document.querySelectorAll('[data-testid=\"backup-plaintext-warning\"]').length")" = 1 ] || fail "the warning about secrets in plain text is not shown"
admin screenshot --full "$PRINTS/01-backup-tab.png" >/dev/null
admin download "$(testid backup-download)" "$WORK/plain.json" >/dev/null || fail "the plain backup was not downloaded"
python3 - "$WORK/plain.json" "$PUSH_TOKEN" <<'PY' || fail "the plain backup does not have the registered items"
import json, sys
backup = json.load(open(sys.argv[1]))
assert backup["format"] == "gatus-admin-backup"
assert [e["key"] for e in backup["endpoints"]] == ["jobs_backup"] and sys.argv[2] in backup["endpoints"][0]["definition"]
assert [p["slug"] for p in backup["statusPages"]] == ["jobs"] and [k["name"] for k in backup["pushKeys"]] == ["akamai"]
PY
admin click "$(testid backup-encrypt)" >/dev/null
admin fill "$(testid backup-password)" "$BACKUP_PASSWORD" >/dev/null
admin fill "$(testid backup-password-confirm)" "$BACKUP_PASSWORD" >/dev/null
admin screenshot --full "$PRINTS/02-backup-encrypted.png" >/dev/null
admin download "$(testid backup-download)" "$WORK/encrypted.json" >/dev/null || fail "the encrypted backup was not downloaded"
grep -q '"gatus-admin-backup-encrypted"' "$WORK/encrypted.json" || fail "the backup is not encrypted"
grep -q "$PUSH_TOKEN" "$WORK/encrypted.json" && fail "the encrypted backup has the token in plain text"
admin close >/dev/null 2>&1 || true
stop_gatus

step "Target installation: preview with the status page of the file skipped"
start_gatus target
login_screen
admin open "$BASE/admin/backup" >/dev/null
admin wait "$(testid restore-file)" >/dev/null || fail "the restore section did not open"
admin upload "$(testid restore-file)" "$WORK/plain.json" >/dev/null
admin wait 500 >/dev/null
admin click "$(testid restore-preview)" >/dev/null
admin wait "$(testid restore-plan-table)" >/dev/null || fail "the preview was not shown"
[ "$(plan_action pushKey akamai)" = create ] || fail "the push key is not planned to be created"
[ "$(plan_action endpoint jobs_backup)" = create ] || fail "the endpoint is not planned to be created"
[ "$(plan_action statusPage jobs)" = skip ] || fail "the status page of the configuration file is not skipped"
authenticated "$BASE/api/v1/admin/endpoints" | grep -q jobs_backup && fail "the preview created the endpoint"
admin wait 500 >/dev/null
admin screenshot --full "$PRINTS/03-restore-preview.png" >/dev/null

step "Options invalidate the preview"
admin click "$(testid restore-overwrite)" >/dev/null
admin wait 300 >/dev/null
[ "$(js "document.querySelector('[data-testid=\"restore-apply\"]').disabled")" = true ] || fail "the restore is not disabled after changing an option"
admin click "$(testid restore-overwrite)" >/dev/null
admin click "$(testid restore-preview)" >/dev/null
admin wait "$(testid restore-plan-table)" >/dev/null

step "Restore with confirmation and results"
admin click "$(testid restore-apply)" >/dev/null
admin wait "$(testid confirm-accept)" >/dev/null || fail "the confirmation was not shown"
admin click "$(testid confirm-accept)" >/dev/null
admin wait "$(testid restore-results-table)" >/dev/null || fail "the results were not shown"
[ "$(result_of pushKey akamai)" = created ] && [ "$(result_of endpoint jobs_backup)" = created ] && [ "$(result_of statusPage jobs)" = skipped ] || fail "unexpected results"
admin wait 500 >/dev/null
admin screenshot --full "$PRINTS/04-restore-results.png" >/dev/null
curl -s "$BASE/api/push/$KEY_TOKEN/jobs_backup?status=up&msg=restored" | grep -q '"ok":true' || fail "the restored push key does not accept the original token"
curl -s "$BASE/api/push/$PUSH_TOKEN?status=up&msg=restored" | grep -q '"ok":true' || fail "the restored push endpoint does not accept its token"

step "The same restore again leaves the items unchanged"
admin click "$(testid restore-preview)" >/dev/null
admin wait "$(testid restore-plan-table)" >/dev/null
for _ in $(seq 1 10); do
  [ "$(plan_action endpoint jobs_backup)" = unchanged ] && break
  sleep 0.3
done
[ "$(plan_action pushKey akamai)" = unchanged ] && [ "$(plan_action endpoint jobs_backup)" = unchanged ] || fail "the applied items are not unchanged"

step "Encrypted backup: wrong password, then the right one, in dark mode"
admin set media dark >/dev/null
admin reload >/dev/null
admin wait "$(testid restore-file)" >/dev/null
admin upload "$(testid restore-file)" "$WORK/encrypted.json" >/dev/null
admin wait "$(testid restore-password)" >/dev/null || fail "the password of the encrypted backup is not asked"
admin fill "$(testid restore-password)" "a wrong password" >/dev/null
admin click "$(testid restore-preview)" >/dev/null
admin wait "$(testid restore-error)" >/dev/null || fail "the wrong password did not show an error"
grep -qi "invalid password" <<<"$(js "document.querySelector('[data-testid=\"restore-error\"]').innerText")" || fail "unexpected error of the wrong password"
admin fill "$(testid restore-password)" "$BACKUP_PASSWORD" >/dev/null
admin click "$(testid restore-preview)" >/dev/null
admin wait "$(testid restore-plan-table)" >/dev/null || fail "the encrypted backup was not previewed"
[ "$(plan_action endpoint jobs_backup)" = unchanged ] || fail "the encrypted backup is not the same as the plain one"
admin mouse move 5 5 >/dev/null 2>&1 || true
admin wait 800 >/dev/null
admin screenshot --full "$PRINTS/05-restore-encrypted-dark.png" >/dev/null
admin set media light >/dev/null

echo "OK: $STEP steps; screenshots in $PRINTS"
