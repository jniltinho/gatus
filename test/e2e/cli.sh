#!/usr/bin/env bash
# End-to-end tests of the command line, against the locally built binary.
#
#   test/e2e/cli.sh
#
# Checks the commands that do not need a server (version, config validate, password hash), then starts the server
# WITHOUT a command — as the ENTRYPOINT of the image does — with --config while the environment points to another file,
# and checks that a reload loads the file of the flag again, that an invalid update keeps the current configuration and
# the process, and that SIGTERM after a reload stops cleanly. It takes about a minute and a half, because the
# configuration is only checked every 30 seconds.
set -uo pipefail
ROOT=$(git rev-parse --show-toplevel)
cd "$ROOT"
WORK=$(mktemp -d)
PORT=${E2E_PORT:-18083}
PID=""
# The binary of THIS run, in the temporary directory: a build that fails must never leave an older "$GATUS" to be
# tested in its place
GATUS="$WORK/gatus"
cleanup() {
  if [ -n "$PID" ] && kill -0 "$PID" 2>/dev/null; then
    kill -TERM "$PID" 2>/dev/null
    for _ in $(seq 1 10); do kill -0 "$PID" 2>/dev/null || break; sleep 1; done
    kill -KILL "$PID" 2>/dev/null
    wait "$PID" 2>/dev/null
  fi
  rm -rf "$WORK"
}
trap cleanup EXIT
fail() { echo "FAILED: $*"; [ -f "$WORK/gatus.log" ] && tail -15 "$WORK/gatus.log"; exit 1; }
get() { curl -s --max-time 5 "$@"; }
echo "==> Building"
make -s build BINARY=gatus DIST="$WORK" || fail "the build failed"
[ -x "$GATUS" ] || fail "the build did not produce $GATUS"
# Written to a temporary file and renamed, so that the server never reads a half-written configuration
write() { cat > "$WORK/flag.yaml.tmp" <<CONFIG
web: {address: 127.0.0.1, port: $PORT}
skip-invalid-config-update: true
endpoints:
  - name: $1
    url: http://127.0.0.1:$PORT/health
    interval: 1h
    conditions: ["[STATUS] == 200"]
CONFIG
  mv "$WORK/flag.yaml.tmp" "$WORK/flag.yaml"
}
write first
"$GATUS" --help | grep -q "healthcheck" || fail "the help does not list the commands"
"$GATUS" version | grep -q "^gatus " || fail "gatus version"
"$GATUS" config validate --config "$WORK/flag.yaml" | grep -q "The configuration is valid" || fail "config validate refused a valid configuration"
"$GATUS" config validate --config "$WORK/typo.yaml" >/dev/null 2>&1 && fail "config validate accepted a path that does not exist"
printf 'endpoints:\n  - name: broken\n    url: http://127.0.0.1\n' > "$WORK/invalid.yaml"
"$GATUS" config validate --config "$WORK/invalid.yaml" >/dev/null 2>&1 && fail "config validate accepted an endpoint without conditions"
hash=$(printf 'a-good-password\n' | "$GATUS" password hash) || fail "password hash"
[ "$(printf '%s' "$hash" | base64 -d 2>/dev/null | cut -c1-4)" = '$2a$' ] || fail "password hash did not print a bcrypt hash in base64: $hash"
refusal=$("$GATUS" password hash a-good-password 2>&1) && fail "password hash accepted the password as an argument"
grep -q "a-good-password" <<<"$refusal" && fail "the refusal of the argument repeats the password: $refusal"
"$GATUS" healthcheck --url "http://127.0.0.1:$PORT/health" >/dev/null 2>&1 && fail "healthcheck was healthy with no server"
echo "0. version, config validate, password hash and healthcheck without a server"
# The environment points to ANOTHER file: a reload must not go back to it
printf 'web: {address: 127.0.0.1, port: %s}\nendpoints:\n  - name: from-environment\n    url: http://127.0.0.1:%s/health\n    interval: 1h\n    conditions: ["[STATUS] == 200"]\n' "$PORT" "$PORT" > "$WORK/env.yaml"
# Without a command, as the ENTRYPOINT of the image runs it
GATUS_CONFIG_PATH="$WORK/env.yaml" "$GATUS" --config "$WORK/flag.yaml" --log-level INFO > "$WORK/gatus.log" 2>&1 &
PID=$!
for _ in $(seq 1 30); do
  kill -0 "$PID" 2>/dev/null || fail "the server died while starting"
  get -f "http://127.0.0.1:$PORT/health" >/dev/null && break
  sleep 1
done
get -f "http://127.0.0.1:$PORT/health" >/dev/null || fail "the server did not answer /health"
names() { get "http://127.0.0.1:$PORT/api/v1/endpoints/statuses" | python3 -c "import json,sys; print(','.join(e['name'] for e in json.load(sys.stdin)))"; }
sleep 3
[ "$(names)" = first ] || fail "the server did not start with the file of the flag: $(names)"
echo "1. without a command, started with the file of --config and not the one of the environment"
"$GATUS" healthcheck --config "$WORK/flag.yaml" >/dev/null || fail "healthcheck of the running server"
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
PID=""
grep -q "Shutting down" "$WORK/gatus.log" || fail "no clean shutdown in the log"
[ "$code" = 0 ] || fail "exit code $code after SIGTERM"
echo "5. SIGTERM after a reload stops cleanly (exit 0)"
echo "OK: command line end-to-end tests passed"
