#!/usr/bin/env bash
# Upgrade test between two published images, with Docker: a backup made on the old version is restored on the new one,
# the new version is started over the database of the old one, and the Python scripts of docs/ run against both.
#
#   test/e2e/upgrade.sh                                                  # v5.36.0-fork.27 -> v6.0.0
#   OLD_IMAGE=jniltinho/gatus:v6.0.0 NEW_IMAGE=gatus:dev test/e2e/upgrade.sh
#
# Run it before a release, with NEW_IMAGE built from the candidate. Requires Docker, curl and python3 with bcrypt.
set -euo pipefail

REPO=$(cd "$(dirname "$0")/../.." && pwd)
WORK=$(mktemp -d)
OLD_IMAGE=${OLD_IMAGE:-jniltinho/gatus:v5.36.0-fork.27}
NEW_IMAGE=${NEW_IMAGE:-jniltinho/gatus:v6.0.0}
USERNAME=admin
PASSWORD='upgrade-test-password'
BACKUP_PASSWORD='upgrade-backup-password'
SUFFIX=$$

mkdir -p "$WORK"/{old,new,inplace}
# The containers run as the user of this script (see start), so that it can remove what they write

passed=0
ok() { passed=$((passed + 1)); echo "  ok: $1"; }
fail() { echo "  FAILED: $1"; exit 1; }
step() { echo "==> $1"; }
# The containers are found by name and not kept in a variable: start runs in a command substitution, which is a
# subshell, and what it appends to an array never reaches this shell.
cleanup() {
  docker ps -aq --filter "name=^gatus-upgrade-(old|new|inplace)-$SUFFIX\$" | xargs -r docker rm -f >/dev/null 2>&1 || true
  rm -rf "$WORK"
}
trap cleanup EXIT

step "generate-admin-password.py and gatus password hash both hash the login password"
HASH=$(printf '%s' "$PASSWORD" | python3 "$REPO/docs/generate-admin-password.py" --stdin)
[ -n "$HASH" ] && ok "hash from the Python script (${#HASH} characters)"
python3 - "$HASH" "$PASSWORD" <<'PY' && ok "the hash of the script is a valid bcrypt in base64" || fail "invalid hash"
import base64, sys
raw = base64.urlsafe_b64decode(sys.argv[1] + "=" * (-len(sys.argv[1]) % 4)).decode()
assert raw.startswith("$2") and len(raw) == 60, raw[:4]
PY
CLI_HASH=$(printf '%s\n' "$PASSWORD" | docker run --rm -i "$NEW_IMAGE" password hash)
[ -n "$CLI_HASH" ] && [ "$CLI_HASH" != "$HASH" ] && ok "gatus password hash of the new image gives another hash (own salt) of the same password"

write_config() { # directory hash
  cat > "$1/config.yaml" <<CONFIG
web:
  port: 8080
storage:
  type: sqlite
  path: /data/data.db
security:
  basic:
    username: $USERNAME
    password-bcrypt-base64: "$2"
admin:
  enabled: true
status-pages:
  enabled: true
endpoints:
  - name: health
    group: core
    url: http://127.0.0.1:8080/health
    interval: 30s
    conditions:
      - "[STATUS] == 200"
CONFIG
}

start() { # name image directory -> prints the base URL
  local name="gatus-upgrade-$1-$SUFFIX"
  docker run -d --name "$name" --user "$(id -u):$(id -g)" -p 127.0.0.1::8080 -v "$3:/data" -v "$3/config.yaml:/config/config.yaml:ro" "$2" >/dev/null
  local port
  port=$(docker port "$name" 8080/tcp | head -1 | sed 's/.*://')
  for _ in $(seq 1 60); do
    if curl -fsS "http://127.0.0.1:$port/health" >/dev/null 2>&1; then echo "http://127.0.0.1:$port"; return 0; fi
    sleep 0.5
  done
  docker logs "$name" 2>&1 | tail -20 >&2
  return 1
}

manager() { # base subcommand args...
  local base=$1; shift
  python3 "$REPO/docs/manager-gatus.py" "$@" --gatus-url "$base" --username "$USERNAME" --password "$PASSWORD"
}
api() { # base method path [curl args...]
  local base=$1 method=$2 path=$3; shift 3
  curl -sS -u "$USERNAME:$PASSWORD" -X "$method" -H 'X-Requested-With: XMLHttpRequest' "$@" "$base$path"
}

# The Python hash is the login of the old version, the hash of the CLI the login of the new one: both must work
write_config "$WORK/old" "$HASH"
write_config "$WORK/new" "$CLI_HASH"

step "Old version: endpoints through manager-gatus.py, a status page with a login and a push key"
OLD=$(start old "$OLD_IMAGE" "$WORK/old") || fail "the old version did not start"
ok "old version up at $OLD, login with the hash of the Python script: $(api "$OLD" GET /api/v1/admin/metadata -o /dev/null -w '%{http_code}')"
cat > "$WORK/endpoints.csv" <<'CSV'
grupo,nome,url
shop,checkout,https://example.org/checkout
shop,catalog,https://example.org/catalog
payments,gateway,https://example.com/gateway
CSV
manager "$OLD" import --csv "$WORK/endpoints.csv" > "$WORK/import.log" 2>&1 || { cat "$WORK/import.log"; fail "import"; }
count=$(api "$OLD" GET /api/v1/admin/endpoints | python3 -c 'import json,sys; d=json.load(sys.stdin); print(sum(1 for i in d if i["source"]=="admin"))')
[ "$count" = 3 ] && ok "import: 3 managed endpoints, with push" || fail "expected 3 endpoints, found $count"
manager "$OLD" rename-group --from shop --to Storefront > "$WORK/rename.log" 2>&1 || { cat "$WORK/rename.log"; fail "rename-group"; }
manager "$OLD" groups > "$WORK/groups-old.txt" 2>&1
grep -qi "storefront" "$WORK/groups-old.txt" && ! grep -qiw "shop" "$WORK/groups-old.txt" && ok "rename-group: shop -> Storefront" || { cat "$WORK/groups-old.txt"; fail "rename-group did not rename"; }
manager "$OLD" export-tokens --out "$WORK/tokens-old.csv" > /dev/null 2>&1 || fail "export-tokens"
[ "$(wc -l < "$WORK/tokens-old.csv")" = 4 ] && ok "export-tokens: header + 3 tokens" || { cat "$WORK/tokens-old.csv"; fail "tokens"; }

PAGE_HASH=$(printf '%s' 'status-page-password' | python3 "$REPO/docs/generate-admin-password.py" --stdin)
cat > "$WORK/page.json" <<JSON
{"slug":"customers","title":"Customers","enabled":true,"groups":["Storefront"],"auth":{"username":"customer","password-bcrypt-base64":"$PAGE_HASH"}}
JSON
code=$(api "$OLD" POST /api/v1/admin/status-pages -H 'Content-Type: application/json' --data @"$WORK/page.json" -o "$WORK/page-created.json" -w '%{http_code}')
[ "$code" = 201 ] && ok "status page with a login created (201)" || { cat "$WORK/page-created.json"; fail "status page: $code"; }
code=$(api "$OLD" POST /api/v1/admin/push-keys -H 'Content-Type: application/json' --data '{"name":"scripts"}' -o "$WORK/key.json" -w '%{http_code}')
[ "$code" = 201 ] && ok "global push key created (201)" || { cat "$WORK/key.json"; fail "push key: $code"; }
GLOBAL_TOKEN=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["token"])' "$WORK/key.json")
FIRST_TOKEN=$(sed -n 2p "$WORK/tokens-old.csv" | cut -d, -f2)
FIRST_HOST=$(sed -n 2p "$WORK/tokens-old.csv" | cut -d, -f1)

step "Old version: backup without and with a password"
api "$OLD" POST /api/v1/admin/backup -H 'Content-Type: application/json' --data '{}' -o "$WORK/backup.json"
api "$OLD" POST /api/v1/admin/backup -H 'Content-Type: application/json' --data "{\"password\":\"$BACKUP_PASSWORD\"}" -o "$WORK/backup.enc.json"
python3 - "$WORK/backup.json" "$WORK/backup.enc.json" <<'PY' && ok "backup: 3 endpoints, 1 status page, 1 push key; the encrypted one exposes nothing" || fail "content of the backup"
import json, sys
plain = json.load(open(sys.argv[1])); encrypted = open(sys.argv[2]).read()
assert len(plain["endpoints"]) == 3 and len(plain["statusPages"]) == 1 and len(plain["pushKeys"]) == 1, {k: len(v) for k, v in plain.items() if isinstance(v, list)}
assert "example.org" not in encrypted and "customers" not in encrypted and "aes-256-gcm" in encrypted
PY

restore() { # base file password overwrite -> prints the result JSON
  local base=$1 file=$2 password=$3 overwrite=$4
  python3 - "$file" "$password" "$overwrite" > "$WORK/restore.json" <<'PY'
import json, sys
print(json.dumps({"file": json.load(open(sys.argv[1])), "password": sys.argv[2], "overwrite": sys.argv[3] == "true"}))
PY
  api "$base" POST /api/v1/admin/restore/preview -H 'Content-Type: application/json' --data @"$WORK/restore.json" -o "$WORK/preview.json"
  python3 - "$WORK/restore.json" "$WORK/preview.json" > "$WORK/apply.json" <<'PY'
import json, sys
request = json.load(open(sys.argv[1])); preview = json.load(open(sys.argv[2]))
assert "fingerprint" in preview, preview
request["fingerprint"] = preview["fingerprint"]
print(json.dumps(request))
PY
  api "$base" POST /api/v1/admin/restore -H 'Content-Type: application/json' --data @"$WORK/apply.json"
}
summary() { python3 -c 'import json,sys; s=json.load(sys.stdin)["summary"]; print(" ".join(f"{k}={s[k]}" for k in sorted(s)))'; }

step "New version, empty: restores the ENCRYPTED backup of the old version"
NEW=$(start new "$NEW_IMAGE" "$WORK/new") || fail "the new version did not start"
ok "new version up at $NEW, login with the hash of gatus password hash: $(api "$NEW" GET /api/v1/admin/metadata -o /dev/null -w '%{http_code}')"
code=$(python3 - "$WORK/backup.enc.json" <<'PY' | curl -sS -u "$USERNAME:$PASSWORD" -H 'X-Requested-With: XMLHttpRequest' -H 'Content-Type: application/json' --data @- -o /dev/null -w '%{http_code}' "$NEW/api/v1/admin/restore/preview"
import json, sys
print(json.dumps({"file": json.load(open(sys.argv[1])), "password": "wrong-on-purpose-password", "overwrite": False}))
PY
)
[ "$code" = 400 ] && ok "wrong password of the backup: 400" || fail "the wrong password answered $code"
result=$(restore "$NEW" "$WORK/backup.enc.json" "$BACKUP_PASSWORD" false)
echo "$result" > "$WORK/result-new.json"
line=$(echo "$result" | summary)
echo "$line" | grep -q "created=5" && echo "$line" | grep -q "failed=0" && ok "restore: $line" || { echo "$result" | head -c 600; fail "restore on the new version: $line"; }

step "New version: what was restored is what the old version had"
manager "$NEW" endpoints > "$WORK/endpoints-new.txt" 2>&1; manager "$OLD" endpoints > "$WORK/endpoints-old.txt" 2>&1
diff <(grep -iE "storefront|payments" "$WORK/endpoints-old.txt" | sort) <(grep -iE "storefront|payments" "$WORK/endpoints-new.txt" | sort) >/dev/null && ok "manager-gatus.py endpoints: same listing on both versions" || { diff "$WORK/endpoints-old.txt" "$WORK/endpoints-new.txt" | head; fail "the listings differ"; }
manager "$NEW" status-pages > "$WORK/pages-new.txt" 2>&1
grep -qi "customers" "$WORK/pages-new.txt" && ok "manager-gatus.py status-pages: the customers page exists on the new version" || { cat "$WORK/pages-new.txt"; fail "status page missing"; }
manager "$NEW" export-tokens --out "$WORK/tokens-new.csv" > /dev/null 2>&1
diff <(sort "$WORK/tokens-old.csv") <(sort "$WORK/tokens-new.csv") >/dev/null && ok "export-tokens: the push tokens are the same after the restore" || fail "the tokens changed"
code=$(curl -sS -o /dev/null -w '%{http_code}' "$NEW/api/push/$FIRST_TOKEN?status=up&msg=after-the-restore&ping=12")
[ "$code" = 200 ] && ok "push with the original token of the endpoint ($FIRST_HOST): 200" || fail "push of the endpoint: $code"
key=$(api "$NEW" GET /api/v1/admin/endpoints | python3 -c 'import json,sys; print([i["key"] for i in json.load(sys.stdin) if i["source"]=="admin"][0])')
code=$(curl -sS -o /dev/null -w '%{http_code}' "$NEW/api/push/$GLOBAL_TOKEN/$key?status=up")
[ "$code" = 200 ] && ok "push with the original global key: 200" || fail "push with the global key: $code"
code=$(curl -sS -o /dev/null -w '%{http_code}' "$NEW/api/v1/status-pages/customers")
[ "$code" = 401 ] && ok "customers page without credentials: 401" || fail "page without credentials: $code"
code=$(curl -sS -o "$WORK/page-public.json" -w '%{http_code}' -u customer:status-page-password "$NEW/api/v1/status-pages/customers")
[ "$code" = 200 ] && ok "customers page with the original login: 200" || fail "login of the page: $code"
python3 - "$WORK/page-public.json" <<'PY' && ok "the public page lists the 2 endpoints of the group, with the counts" || fail "content of the page"
import json, sys
page = json.load(open(sys.argv[1]))
names = sorted(e["name"] for g in page["groups"] for e in g["endpoints"])
assert names == ["catalog", "checkout"], names
assert page["summary"]["total"] == 2, page["summary"]
PY

step "New version: the same restore again changes nothing, and neither does the plain backup with overwrite"
line=$(restore "$NEW" "$WORK/backup.enc.json" "$BACKUP_PASSWORD" false | summary)
echo "$line" | grep -q "created=0" && echo "$line" | grep -q "failed=0" && ok "again: $line" || fail "again: $line"
line=$(restore "$NEW" "$WORK/backup.json" "" true | summary)
echo "$line" | grep -q "failed=0" && echo "$line" | grep -q "created=0" && ok "plain backup with overwrite: $line" || fail "overwrite: $line"

step "The backup of the new version has the format of the old one"
api "$NEW" POST /api/v1/admin/backup -H 'Content-Type: application/json' --data '{}' -o "$WORK/backup-v6.json"
python3 - "$WORK/backup.json" "$WORK/backup-v6.json" <<'PY' && ok "same format version and same definitions as the backup of the old version" || fail "format of the backup"
import json, sys
old, new = (json.load(open(p)) for p in sys.argv[1:3])
assert old["version"] == new["version"], (old["version"], new["version"])
strip = lambda items: sorted(json.dumps(i.get("definition", i), sort_keys=True) for i in items)
assert strip(old["endpoints"]) == strip(new["endpoints"])
assert strip(old["statusPages"]) == strip(new["statusPages"])
PY

step "In-place upgrade: the new version starts over the database of the old one"
old_name="gatus-upgrade-old-$SUFFIX"
docker stop "$old_name" >/dev/null
cp "$WORK/old/data.db"* "$WORK/inplace/" 2>/dev/null
write_config "$WORK/inplace" "$HASH"
INPLACE=$(start inplace "$NEW_IMAGE" "$WORK/inplace") || fail "the new version did not start over the database of the old one"
manager "$INPLACE" endpoints > "$WORK/endpoints-inplace.txt" 2>&1
diff <(grep -iE "storefront|payments" "$WORK/endpoints-old.txt" | sort) <(grep -iE "storefront|payments" "$WORK/endpoints-inplace.txt" | sort) >/dev/null && ok "the 3 managed endpoints are still there" || { cat "$WORK/endpoints-inplace.txt"; fail "the endpoints are gone"; }
code=$(curl -sS -o /dev/null -w '%{http_code}' -u customer:status-page-password "$INPLACE/api/v1/status-pages/customers")
[ "$code" = 200 ] && ok "the page with a login still opens: 200" || fail "status page: $code"
# Right after the start, /health already answers while the managed endpoints are still being loaded, and a push gets
# 404 for a second or two (v5.36.0-fork.27 does the same): a pusher retries, and so does this check
attempts=0; code=000
for _ in $(seq 1 20); do
  attempts=$((attempts + 1))
  code=$(curl -sS -o /dev/null -w '%{http_code}' "$INPLACE/api/push/$FIRST_TOKEN?status=up")
  [ "$code" = 200 ] && break
  sleep 0.5
done
[ "$code" = 200 ] && ok "the push with the original token is still accepted: 200 (attempt $attempts)" || fail "push: $code after $attempts attempts"
health=$(docker inspect "gatus-upgrade-inplace-$SUFFIX" --format '{{.State.Health.Status}}')
for _ in $(seq 1 40); do [ "$health" = healthy ] && break; sleep 1; health=$(docker inspect "gatus-upgrade-inplace-$SUFFIX" --format '{{.State.Health.Status}}'); done
[ "$health" = healthy ] && ok "HEALTHCHECK of the new image: healthy" || fail "healthcheck: $health"
errors=$(docker logs "gatus-upgrade-inplace-$SUFFIX" 2>&1 | grep -ciE "panic|\[ERROR\]|level=error" || true)
[ "$errors" = 0 ] && ok "no error nor panic in the log of the new version" || { docker logs "gatus-upgrade-inplace-$SUFFIX" 2>&1 | grep -iE "panic|error" | head -5; fail "$errors errors in the log"; }

echo "OK: $passed checks"
