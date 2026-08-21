# omniroute-cli

Current release: **v0.6.1**

`omniroute-cli` is a Go operations CLI and Docker Compose stack for running **OmniRoute** together with **Open WebUI**. The Go CLI is the authoritative runtime control plane; the `justfile` and `update-cli.yam` delegate operational behavior to it to avoid lifecycle drift.

## Quick start

```bash
just build
just init
just install
omniroute-cli up
omniroute-cli status
```

`just install` installs to `/usr/local/bin/omniroute-cli` by default. Verify the binary actually used by your shell with:

```bash
command -v omniroute-cli
omniroute-cli --version
```

If an older installed binary still looks for `compose.yaml`, rebuild and reinstall the current CLI with `just install`. Since v0.6.1, the CLI resolves standard Compose filenames compatibly in this order: `docker-compose.yaml`, `docker-compose.yml`, `compose.yaml`, `compose.yml`. A custom `--compose-file` path remains authoritative and fails if that file is missing.

## Semantic Versioning

Project and build versions follow Semantic Versioning. Runtime code does not compare or sort versions as raw strings: the CLI uses its own SemVer parser/type (`src/semver`) for `major.minor.patch`, prerelease identifiers, build metadata, and precedence comparisons.

`VERSION` currently contains:

```text
0.6.1
```

Build scripts validate it through the same Go SemVer implementation:

```bash
go run ./cmd/semver-check "$(cat VERSION)"
```

## Secure initialization

`.env` is never committed. `omniroute-cli init` reads `.env.example`, replaces generated placeholders with cryptographically random values, replaces OmniRoute's public initial password `123456` with a unique password, and writes `.env` with mode `0600`.

Services bind to localhost by default:

```dotenv
PUBLIC_BIND_ADDRESS=127.0.0.1
OMNIROUTE_PUBLIC_PORT=20128
OPENWEBUI_PUBLIC_PORT=20000
```

`doctor` and `secrets check` flag dangerous combinations such as public binding with API-key enforcement disabled or insecure cookies.

## Global options

Global options may be placed before or after the command:

```text
--project-dir DIR       project directory, default .
--compose-file FILE     auto-detect; prefers docker-compose.yaml
--project-name NAME     Compose project/container/volume prefix
--prefix NAME           alias for --project-name
--timeout DURATION      per-operation timeout, default 2m
--json                  machine-readable JSON output
--dry-run               print Docker operations without executing
--version, -v           semantic CLI version
```

Examples:

```bash
omniroute-cli status --json
omniroute-cli update --timeout 15m
omniroute-cli --project-name omniroute-dev up
```

## Lifecycle commands

```text
omniroute-cli init
omniroute-cli up | run
omniroute-cli pull
omniroute-cli update [--plan] [--wait 2m]
omniroute-cli rollback [--previous]
omniroute-cli recreate
omniroute-cli start [SERVICE...]
omniroute-cli stop [SERVICE...]
omniroute-cli restart [SERVICE...]
omniroute-cli down
omniroute-cli clean
omniroute-cli clean-all --yes
omniroute-cli prune
```

### Start versus update

`up` does **not** pull from a registry. It only makes the configured stack run:

```bash
docker compose up -d --remove-orphans
```

Use `pull` explicitly to fetch images, or `update` for the complete update path.

### Transactional image update and rollback

Preview without mutation:

```bash
omniroute-cli update --plan
```

An update performs:

```text
validate configuration
capture current immutable image digests
store .omniroute-cli/rollback.json
pull configured images
force-recreate containers
wait for HTTP health
success -> keep new images
failure -> recreate previous digests automatically
```

Manual rollback:

```bash
omniroute-cli rollback --previous
```

Rollback state is local and ignored by Git. No application-data backup or restore feature is implemented in the Go CLI.

## Health, readiness and diagnostics

```bash
omniroute-cli health
omniroute-cli health --deep
omniroute-cli health --wait 2m
omniroute-cli doctor
omniroute-cli doctor --deep
omniroute-cli status
omniroute-cli status --json
```

OmniRoute health uses `/api/monitoring/health`. When management authentication is enabled, set an appropriate management credential:

```dotenv
OMNIROUTE_MANAGEMENT_TOKEN=oma_live_...
```

For least privilege, use a scoped OmniRoute Access Token. Ordinary inference keys do not automatically authorize `/api/*` management routes.

Open WebUI is probed through `/health`; deep checks additionally use `/ready`. Docker Compose also defines container healthchecks, and Open WebUI waits for OmniRoute's container health before startup.

## Runtime information

```text
omniroute-cli status
omniroute-cli top
omniroute-cli images
omniroute-cli resolved-images
omniroute-cli services
omniroute-cli urls
omniroute-cli info
omniroute-cli logs [-f] [--tail N] [SERVICE]
omniroute-cli log [--tail N] [SERVICE]
omniroute-cli shell [SERVICE]
omniroute-cli compose-config
```

Service aliases:

```text
omniroute
openwebui
open-webui
```

## Configuration management

`config` now manages `.env`; raw Compose rendering moved to `compose-config`.

```bash
omniroute-cli config list
omniroute-cli config get OMNIROUTE_PUBLIC_PORT
omniroute-cli config set OMNIROUTE_PUBLIC_PORT 21128
omniroute-cli config validate
omniroute-cli config path
```

Secret values are redacted and cannot be changed through `config set`.

## Secret management

```bash
omniroute-cli secrets check
omniroute-cli secrets rotate jwt
omniroute-cli secrets rotate api
omniroute-cli secrets rotate initial-password
omniroute-cli secrets rotate ws
omniroute-cli secrets rotate openwebui
omniroute-cli secrets rotate --safe --yes
```

Storage encryption rotation is deliberately guarded:

```bash
omniroute-cli secrets rotate storage --yes
```

Changing the storage encryption key can make existing encrypted data unreadable unless a proper migration is performed. `--safe` explicitly excludes the storage key, but can still invalidate sessions/tokens and therefore requires `--yes`.

## OmniRoute API commands

The CLI exposes useful OmniRoute operational endpoints:

```bash
omniroute-cli models list
omniroute-cli providers status
omniroute-cli sessions
omniroute-cli usage
omniroute-cli cache stats
omniroute-cli cache clear --yes
```

`models list` uses the OpenAI-compatible `/v1/models?prefix=alias` endpoint. Management commands use the current OmniRoute `/api/*` management endpoints and `OMNIROUTE_MANAGEMENT_TOKEN` when configured.

All of these can use `--json`.

## Multi-instance operation

`docker-compose.yaml` parameterizes the Compose project, container names, volumes and network through the prefix. The default remains `ai-tools-omniroute`.

Example second installation:

```bash
omniroute-cli --project-dir ./dev --project-name omniroute-dev up
```

The prefix affects Compose project isolation plus container/volume/network names. Use different published ports in each project's `.env`.

## Shell completion

```bash
omniroute-cli completion bash
omniroute-cli completion zsh
omniroute-cli completion fish
```

Redirect the generated completion into the appropriate shell completion location.

## `just` development workflow

```bash
just fmt
just test
just check
just build
just install
```

Operational recipes delegate to the Go CLI:

```bash
just up
just update-plan
just update
just rollback
just status
just health
just health-deep
just doctor
```

The existing `backup`, `backup-omniroute`, and `backup-openwebui` recipes are retained only as developer convenience helpers. Backup/restore is intentionally not part of the Go CLI scope.

## update-cli integration

The schema-version-2 manifest is named exactly:

```text
update-cli.yam
```

The currently verified Update CLI auto-detects only `setup.yaml` / `setup.yml`, so pass the custom filename explicitly:

```bash
update-cli --setup-manifest update-cli.yam
update-cli --setup-manifest update-cli.yam --setup-workflow update
update-cli --setup-manifest update-cli.yam --setup-workflow rebuild
update-cli --setup-manifest update-cli.yam --setup-workflow status
update-cli --setup-manifest update-cli.yam --setup-workflow doctor
```

The manifest no longer duplicates Docker lifecycle behavior. It builds `omniroute-cli` and delegates update/start/stop/status/doctor operations to the binary. The `setup`/`update` workflow also installs the freshly built binary to `/usr/local/bin/omniroute-cli` before updating the stack, preventing an older CLI on `PATH` from surviving a project release.

## CI

GitHub Actions validates:

- SemVer through the Go parser
- `gofmt`
- `go vet`
- `go test -race ./...`
- Linux and macOS tests
- CLI build and smoke tests
- secure `.env` generation
- `docker compose config --quiet`
- `justfile` loading
- `update-cli.yam` YAML syntax
- secret scanning
- `govulncheck`

## Releases

`just release` creates the source ZIP from committed files only:

```text
dist/omniroute-cli-v<SEMVER>.zip
```

A tag matching `v<SEMVER>` triggers the release workflow, which builds:

```text
linux/amd64
linux/arm64
darwin/amd64
darwin/arm64
```

and publishes archives plus SHA-256 checksums to GitHub Releases.
