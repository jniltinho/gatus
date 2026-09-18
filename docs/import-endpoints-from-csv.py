#!/usr/bin/env python3
"""Registers the endpoints of a CSV in the administration of Gatus, with push enabled, and exports their push tokens.

The CSV can be the inventory of hostnames, with the columns of the export of the DNS ("Tipo", "Serviço", "Domínio",
"Ambiente", "Hostname", "Destino / Valor", "Status"), or the simple format this script writes, with the columns
"grupo", "nome" and "url". In both cases every endpoint is created as an HTTP check that also accepts push, with a
token of its own, so that a script or an alerting system can report the status of the service in the format of the
Uptime Kuma (see docs/push-monitoring.md).

Examples:

    # 1. Only rewrites the inventory as the simple CSV, keeping production and one row per hostname
    python3 docs/import-endpoints-from-csv.py --csv urls.csv --write-csv urls-prod.csv

    # 2. Registers what is in the CSV, without touching Gatus first (nothing is sent with --dry-run)
    python3 docs/import-endpoints-from-csv.py --csv urls.csv --gatus-url https://status.example.com \\
        --username admin --password 'the-password' --dry-run

    # 3. Registers for real, then exports the tokens of every endpoint that accepts push
    python3 docs/import-endpoints-from-csv.py --csv urls.csv --gatus-url https://status.example.com \\
        --username admin --password 'the-password'
    python3 docs/import-endpoints-from-csv.py --gatus-url https://status.example.com \\
        --username admin --password 'the-password' --export-tokens export_hosts_token.csv

The password can also come from the environment (GATUS_USERNAME and GATUS_PASSWORD), which keeps it out of the shell
history. Only the standard library is used.
"""

from __future__ import annotations

import argparse
import base64
import csv
import json
import os
import re
import secrets
import ssl
import string
import sys
import unicodedata
import urllib.error
import urllib.parse
import urllib.request

# Columns of the inventory of hostnames
INVENTORY_SERVICE = "Serviço"
INVENTORY_ENVIRONMENT = "Ambiente"
INVENTORY_HOSTNAME = "Hostname"
INVENTORY_STATUS = "Status"

# Columns of the simple CSV this script reads and writes
SIMPLE_COLUMNS = ["grupo", "nome", "url"]

TOKEN_ALPHABET = string.ascii_letters + string.digits
TOKEN_LENGTH = 32


class GatusError(Exception):
    """An answer of Gatus that the script cannot go on with"""


def strip_accents(value: str) -> str:
    return "".join(char for char in unicodedata.normalize("NFD", value) if not unicodedata.combining(char))


def slugify(value: str) -> str:
    """Turns a name into a group of Gatus: lowercase, without accents and with a single hyphen between the words"""
    slug = re.sub(r"[^a-z0-9]+", "-", strip_accents(value).lower())
    return slug.strip("-")


def generate_token() -> str:
    return "".join(secrets.choice(TOKEN_ALPHABET) for _ in range(TOKEN_LENGTH))


def read_rows(path: str, environment: str, keep_every_status: bool) -> list[dict[str, str]]:
    """Reads the CSV and returns one row per endpoint, with the group, the name and the URL"""
    with open(path, newline="", encoding="utf-8-sig") as handle:
        reader = csv.DictReader(handle)
        columns = reader.fieldnames or []
        if INVENTORY_HOSTNAME in columns:
            rows = list(read_inventory(reader, environment, keep_every_status))
        elif all(column in columns for column in SIMPLE_COLUMNS):
            rows = [
                {"grupo": row["grupo"].strip(), "nome": row["nome"].strip(), "url": row["url"].strip()}
                for row in reader
                if row.get("url", "").strip()
            ]
        else:
            raise GatusError(
                f"{path}: expected the columns of the inventory ({INVENTORY_HOSTNAME}, {INVENTORY_SERVICE}, "
                f"{INVENTORY_ENVIRONMENT}) or the simple ones ({', '.join(SIMPLE_COLUMNS)}), found {columns}"
            )
    return deduplicate(rows)


def read_inventory(reader: csv.DictReader, environment: str, keep_every_status: bool):
    """Filters the inventory by environment and by status, and yields the rows of the endpoints"""
    wanted = strip_accents(environment).lower()
    for row in reader:
        hostname = (row.get(INVENTORY_HOSTNAME) or "").strip().rstrip(".")
        if not hostname:
            continue
        if wanted and not strip_accents((row.get(INVENTORY_ENVIRONMENT) or "")).lower().startswith(wanted):
            continue
        if not keep_every_status and (row.get(INVENTORY_STATUS) or "").strip().upper() not in ("", "OK"):
            continue
        service = (row.get(INVENTORY_SERVICE) or "").strip()
        yield {"grupo": slugify(service) or "outros", "nome": hostname, "url": ""}


def deduplicate(rows: list[dict[str, str]]) -> list[dict[str, str]]:
    """Keeps one row per name, because the same hostname can appear more than once in the inventory"""
    seen: dict[str, dict[str, str]] = {}
    for row in rows:
        key = row["nome"].lower()
        if key not in seen:
            seen[key] = row
    return sorted(seen.values(), key=lambda row: (row["grupo"], row["nome"]))


def endpoint_url(row: dict[str, str], scheme: str, path: str) -> str:
    if row["url"]:
        return row["url"]
    return f"{scheme}://{row['nome']}{path}"


def write_csv(path: str, rows: list[dict[str, str]], scheme: str, url_path: str) -> None:
    with open(path, "w", newline="", encoding="utf-8") as handle:
        writer = csv.DictWriter(handle, fieldnames=SIMPLE_COLUMNS)
        writer.writeheader()
        for row in rows:
            writer.writerow({"grupo": row["grupo"], "nome": row["nome"], "url": endpoint_url(row, scheme, url_path)})


class Gatus:
    """The administration API of Gatus, with basic authentication"""

    def __init__(self, base_url: str, username: str, password: str, timeout: float, insecure: bool) -> None:
        self.base_url = base_url.rstrip("/")
        credentials = base64.b64encode(f"{username}:{password}".encode()).decode()
        self.authorization = f"Basic {credentials}"
        self.timeout = timeout
        self.context = ssl._create_unverified_context() if insecure else None

    def request(self, method: str, path: str, body: dict | None = None) -> tuple[int, object]:
        data = json.dumps(body).encode() if body is not None else None
        request = urllib.request.Request(self.base_url + path, data=data, method=method)
        request.add_header("Authorization", self.authorization)
        request.add_header("Accept", "application/json")
        if data is not None:
            request.add_header("Content-Type", "application/json")
        # Origin is not sent on purpose: Gatus only refuses an Origin that does not match the address it answers on,
        # which behind a reverse proxy is not the address used here. A request without Origin is not a CSRF risk.
        try:
            with urllib.request.urlopen(request, timeout=self.timeout, context=self.context) as response:
                return response.status, decode(response.read())
        except urllib.error.HTTPError as error:
            return error.code, decode(error.read())
        except urllib.error.URLError as error:
            raise GatusError(f"{method} {path}: {error.reason}") from error

    def list_endpoints(self) -> list[dict]:
        status, body = self.request("GET", "/api/v1/admin/endpoints")
        if status != 200 or not isinstance(body, list):
            raise GatusError(f"the list of endpoints answered {status}: {body}")
        return body

    def get_endpoint(self, key: str) -> dict:
        status, body = self.request("GET", "/api/v1/admin/endpoints/" + urllib.parse.quote(key, safe=""))
        if status != 200 or not isinstance(body, dict):
            raise GatusError(f"the endpoint {key} answered {status}: {body}")
        return body

    def create_endpoint(self, definition: dict) -> tuple[int, object]:
        return self.request("POST", "/api/v1/admin/endpoints", definition)


def decode(raw: bytes) -> object:
    if not raw:
        return None
    try:
        return json.loads(raw)
    except json.JSONDecodeError:
        return raw.decode(errors="replace").strip()


def describe(answer: object) -> str:
    if isinstance(answer, dict) and "error" in answer:
        return str(answer["error"])
    return str(answer)


def build_definition(row: dict[str, str], arguments: argparse.Namespace) -> dict:
    definition: dict[str, object] = {
        "name": row["nome"],
        "group": row["grupo"],
        "url": endpoint_url(row, arguments.scheme, arguments.path),
        "interval": arguments.interval,
        "conditions": arguments.condition,
    }
    if not arguments.no_push:
        definition["push"] = {"enabled": True, "token": generate_token()}
    return definition


def import_endpoints(gatus: Gatus | None, rows: list[dict[str, str]], arguments: argparse.Namespace) -> int:
    existing: set[str] = set()
    if gatus is not None:
        existing = {item.get("key", "") for item in gatus.list_endpoints()}
    created = skipped = failed = 0
    for row in rows:
        definition = build_definition(row, arguments)
        key = f"{slugify(definition['group'])}_{slugify(definition['name'])}"
        if key in existing:
            skipped += 1
            if arguments.verbose:
                print(f"  = {key}: already registered")
            continue
        if gatus is None:
            created += 1
            print(f"  + {key} -> {definition['url']}")
            continue
        status, answer = gatus.create_endpoint(definition)
        if status == 201:
            created += 1
            if arguments.verbose:
                print(f"  + {key} -> {definition['url']}")
        elif status == 409:
            skipped += 1
            print(f"  = {key}: {describe(answer)}")
        else:
            failed += 1
            print(f"  ! {key}: {status} {describe(answer)}", file=sys.stderr)
    print(f"{created} created, {skipped} already there, {failed} refused")
    return 1 if failed else 0


def export_tokens(gatus: Gatus, path: str) -> int:
    """Writes host,token with the push token of every endpoint that accepts push, like export_hosts_token.csv"""
    exported = []
    for item in gatus.list_endpoints():
        if not item.get("acceptsPush"):
            continue
        detail = gatus.get_endpoint(item["key"])
        token = detail.get("pushToken") or ""
        if not token:
            # The endpoint accepts push through the global keys only
            continue
        host = urllib.parse.urlparse(item.get("url") or "").hostname or item.get("name") or item["key"]
        exported.append((host, token))
    exported.sort()
    with open(path, "w", newline="", encoding="utf-8") as handle:
        writer = csv.writer(handle)
        writer.writerow(["host", "token"])
        writer.writerows(exported)
    print(f"{len(exported)} endpoints exported to {path}")
    return 0


def parse_arguments(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Registers the endpoints of a CSV in Gatus with push enabled, and exports their tokens",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=__doc__,
    )
    parser.add_argument("--csv", help="CSV with the endpoints (inventory of hostnames or the simple format)")
    parser.add_argument("--write-csv", metavar="PATH", help="only writes the filtered CSV in the simple format and exits")
    parser.add_argument("--export-tokens", metavar="PATH", help="only exports host,token of the endpoints that accept push and exits")
    parser.add_argument("--gatus-url", default=os.environ.get("GATUS_URL", "http://127.0.0.1:8080"), help="address of Gatus (default: %(default)s)")
    parser.add_argument("--username", default=os.environ.get("GATUS_USERNAME", "admin"), help="user of the administration (default: %(default)s)")
    parser.add_argument("--password", default=os.environ.get("GATUS_PASSWORD"), help="password of the administration (or GATUS_PASSWORD)")
    parser.add_argument("--environment", default="Produção", help="environment kept from the inventory, by the start of the column (default: %(default)s)")
    parser.add_argument("--every-environment", action="store_true", help="keeps every environment of the inventory")
    parser.add_argument("--every-status", action="store_true", help="keeps the rows whose status is not OK")
    parser.add_argument("--group", action="append", metavar="GROUP", help="only these groups, after the slug (can be repeated)")
    parser.add_argument("--scheme", default="https", choices=["https", "http"], help="scheme of the URL built from the hostname (default: %(default)s)")
    parser.add_argument("--path", default="", help="path added to the URL built from the hostname, e.g. /health")
    parser.add_argument("--interval", default="5m", help="interval between the checks (default: %(default)s)")
    parser.add_argument("--condition", action="append", metavar="CONDITION", help="condition of the endpoint (can be repeated; default: [STATUS] == 200)")
    parser.add_argument("--no-push", action="store_true", help="registers without push: no token is generated")
    parser.add_argument("--limit", type=int, metavar="N", help="registers at most N endpoints")
    parser.add_argument("--dry-run", action="store_true", help="shows what would be registered, without calling Gatus")
    parser.add_argument("--timeout", type=float, default=30.0, help="timeout of each request in seconds (default: %(default)s)")
    parser.add_argument("--insecure", action="store_true", help="does not verify the TLS certificate of Gatus")
    parser.add_argument("--verbose", action="store_true", help="prints one line per endpoint")
    arguments = parser.parse_args(argv)
    if not arguments.condition:
        arguments.condition = ["[STATUS] == 200"]
    if arguments.every_environment:
        arguments.environment = ""
    return arguments


def main(argv: list[str]) -> int:
    arguments = parse_arguments(argv)
    needs_gatus = arguments.export_tokens or (arguments.csv and not arguments.write_csv and not arguments.dry_run)
    if needs_gatus and not arguments.password:
        print("the password of the administration is required (--password or GATUS_PASSWORD)", file=sys.stderr)
        return 2
    gatus = None
    if needs_gatus:
        gatus = Gatus(arguments.gatus_url, arguments.username, arguments.password, arguments.timeout, arguments.insecure)
    if arguments.export_tokens:
        return export_tokens(gatus, arguments.export_tokens)
    if not arguments.csv:
        print("--csv is required, unless only --export-tokens is used", file=sys.stderr)
        return 2
    rows = read_rows(arguments.csv, arguments.environment, arguments.every_status)
    if arguments.group:
        wanted = {slugify(group) for group in arguments.group}
        rows = [row for row in rows if row["grupo"] in wanted]
    if arguments.limit:
        rows = rows[: arguments.limit]
    if not rows:
        print("no endpoint left after the filters", file=sys.stderr)
        return 1
    if arguments.write_csv:
        write_csv(arguments.write_csv, rows, arguments.scheme, arguments.path)
        print(f"{len(rows)} endpoints written to {arguments.write_csv}")
        return 0
    print(f"{len(rows)} endpoints in {arguments.csv}" + (" (dry run)" if arguments.dry_run else f" -> {arguments.gatus_url}"))
    return import_endpoints(gatus, rows, arguments)


if __name__ == "__main__":
    try:
        sys.exit(main(sys.argv[1:]))
    except GatusError as error:
        print(str(error), file=sys.stderr)
        sys.exit(1)
    except KeyboardInterrupt:
        sys.exit(130)
