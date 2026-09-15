# Third-party modules kept in the repository

The modules below are used by Gatus and kept here, with their licenses, so that building the fork does not download
them. `go.mod` points to these folders with `replace` directives, so the import paths do not change
(`github.com/TwiN/logr`, for example) and the code stays identical to the upstream.

| Module | Version | License | Folder |
|--------|---------|---------|--------|
| `github.com/TwiN/deepmerge` | v0.2.2 | MIT | `github.com/TwiN/deepmerge` |
| `github.com/TwiN/g8/v2` | v2.0.0 | MIT | `github.com/TwiN/g8` |
| `github.com/TwiN/gocache/v2` | v2.4.0 | MIT | `github.com/TwiN/gocache` |
| `github.com/TwiN/health` | v1.6.0 | MIT | `github.com/TwiN/health` |
| `github.com/TwiN/logr` | v0.3.1 | MIT | `github.com/TwiN/logr` |
| `github.com/TwiN/whois` | v1.3.0 | MIT | `github.com/TwiN/whois` |

Each folder is an unmodified copy of the module at the version above, including its `go.mod`, its `LICENSE` and its
tests. The folders are separate modules, so `go test ./...` at the root of the repository does not run their tests.

## Updating a module

When the upstream of Gatus requires a new version of one of these modules:

```bash
go mod download github.com/TwiN/logr@v0.3.2
rm -rf third_party/github.com/TwiN/logr
cp -r "$(go env GOMODCACHE)/github.com/!twi!n/logr@v0.3.2" third_party/github.com/TwiN/logr
chmod -R u+w third_party/github.com/TwiN/logr
go mod tidy
```

For modules with a major version suffix, the cache folder includes it (e.g. `g8/v2@v2.0.1`), but the folder here does
not (`third_party/github.com/TwiN/g8`). Update the version in the table above and in the `require` of `go.mod`.
