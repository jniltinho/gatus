#!/usr/bin/env bash
# End-to-end tests of the command line, against the locally built binary.
#
#   test/e2e/cli.sh
#
# Checks the commands that do not need a server (version, config validate, password hash), then starts the server with
# --config while the environment points to another file, and checks that a reload loads the file of the flag again,
# that an invalid update keeps the current configuration and the process, and that SIGTERM after a reload stops
# cleanly. It takes about a minute and a half, because the configuration is only checked every 30 seconds.
set -uo pipefail
ROOT=$(git rev-parse --show-toplevel)
cd "$ROOT"
WORK=$(mktemp -d); PORT=${E2E_PORT:-18083}
echo "==> Building"
make -s build
write() { cat > "$WORK/flag.yaml" <<CONFIG
web: {address: 127.0.0.1, port: $PORT}
skip-invalid-config-update: true
endpoints:
  - name: $1
    url: http://127.0.0.1:$PORT/health
    interval: 1h
    conditions: ["[STATUS] == 200"]
CONFIG
}
write first
fail() { echo "FAILED: $*"; [ -f "$WORK/gatus.log" ] && tail -15 "$WORK/gatus.log"; exit 1; }
dist/gatus --help | grep -q "healthcheck" || fail "the help does not list the commands"
dist/gatus version | grep -q "^gatus " || fail "gatus version"
dist/gatus config validate --config "$WORK/flag.yaml" | grep -q "The configuration is valid" || fail "config validate refused a valid configuration"
dist/gatus config validate --config "$WORK/typo.yaml" >/dev/null 2>&1 && fail "config validate accepted a path that does not exist"
printf 'endpoints:\n  - name: broken\n    url: http://127.0.0.1\n' > "$WORK/invalid.yaml"
dist/gatus config validate --config "$WORK/invalid.yaml" >/dev/null 2>&1 && fail "config validate accepted an endpoint without conditions"
hash=$(printf 'a-good-password\n' | dist/gatus password hash) || fail "password hash"
[ "$(printf '%s' "$hash" | base64 -d 2>/dev/null | cut -c1-4)" = '$2a$' ] || fail "password hash did not print a bcrypt hash in base64: $hash"
dist/gatus password hash a-good-password >/dev/null 2>&1 && fail "password hash accepted the password as an argument"
dist/gatus healthcheck --url "http://127.0.0.1:$PORT/health" >/dev/null 2>&1 && fail "healthcheck was healthy with no server"
echo "0. version, config validate, password hash and healthcheck without a server"
# The environment points to ANOTHER file: a reload must not go back to it
printf 'web: {address: 127.0.0.1, port: %s}\nendpoints:\n  - name: from-environment\n    url: http://127.0.0.1:%s/health\n    interval: 1h\n    conditions: ["[STATUS] == 200"]\n' "$PORT" "$PORT" > "$WORK/env.yaml"
GATUS_CONFIG_PATH="$WORK/env.yaml" dist/gatus serve --config "$WORK/flag.yaml" --log-level INFO > "$WORK/gatus.log" 2>&1 &
PID=$!
trap 'kill $PID 2>/dev/null; rm -rf "$WORK"' EXIT
for _ in $(seq 1 30); do curl -sf "http://127.0.0.1:$PORT/health" >/dev/null && break; sleep 1; done
names() { curl -s "http://127.0.0.1:$PORT/api/v1/endpoints/statuses" | python3 -c "import json,sys; print(','.join(e['name'] for e in json.load(sys.stdin)))"; }
sleep 3
[ "$(names)" = first ] || fail "the server did not start with the file of the flag: $(names)"
echo "1. started with the file of --config, not the one of the environment"
dist/gatus healthcheck --config "$WORK/flag.yaml" >/dev/null || fail "healthcheck of the running server"
echo "2. healthcheck reads the port of the configuration"
sleep 1; write second
for _ in $(seq 1 50); do grep -q "Configuration file has been modified" "$WORK/gatus.log" && break; sleep 1; done
for _ in $(seq 1 20); do [ "$(names 2>/dev/null)" = second ] && break; sleep 1; done
[ "$(names)" = second ] || fail "the reload did not load the file of the flag: $(names)"
echo "3. the reload loaded the file of --config again"
sleep 1; echo "this is: [not valid" > "$WORK/flag.yaml"
for _ in $(seq 1 50); do grep -q "The current configuration will continue being used" "$WORK/gatus.log" && break; sleep 1; done
grep -q "The current configuration will continue being used" "$WORK/gatus.log" || fail "the invalid update was not reported"
[ "$(names)" = second ] || fail "the invalid update replaced the configuration"
kill -0 $PID 2>/dev/null || fail "the process died with an invalid update"
echo "4. an invalid update keeps the current configuration and the process"
kill -TERM $PID
for _ in $(seq 1 20); do kill -0 $PID 2>/dev/null || break; sleep 1; done
kill -0 $PID 2>/dev/null && fail "the process did not stop on SIGTERM"
wait $PID; code=$?
grep -q "Shutting down" "$WORK/gatus.log" || fail "no clean shutdown in the log"
[ "$code" = 0 ] || fail "exit code $code after SIGTERM"
echo "5. SIGTERM after a reload stops cleanly (exit 0)"
echo "OK: command line end-to-end tests passed"
