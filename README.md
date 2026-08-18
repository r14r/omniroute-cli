# omniroute-cli

Go CLI and Docker Compose stack for running **OmniRoute** together with **Open WebUI** under the `ai-tools-omniroute` prefix.

## Quick start

```bash
just build
just init
just install
omniroute-cli up
omniroute-cli status
```

`just install` installs to `/usr/local/bin/omniroute-cli` by default. A different directory can be supplied explicitly.

## Secure configuration

`.env` is never committed. On first initialization, `omniroute-cli init` reads `.env.example`, generates unique cryptographically secure values for all secret placeholders, writes `.env` with mode `0600`, and migrates obsolete non-secret defaults.

OmniRoute upstream documents `123456` as its fallback initial dashboard password. This project lists that value only for reference; `omniroute-cli init` generates a unique `OMNIROUTE_INITIAL_PASSWORD` instead.

Services bind to localhost by default:

```dotenv
PUBLIC_BIND_ADDRESS=127.0.0.1
OMNIROUTE_PUBLIC_PORT=20128
OPENWEBUI_PUBLIC_PORT=20000
```

Set `PUBLIC_BIND_ADDRESS=0.0.0.0` only when LAN/public exposure is intentional and appropriate authentication/network controls are in place.

**Upgrade note:** releases before v0.2.0 published fixed example secrets. Existing `.env` files are not auto-rotated because changing storage/session keys can invalidate encrypted data or sessions. `omniroute-cli doctor` detects those known legacy values so they can be rotated deliberately.

## CLI commands

```text
omniroute-cli init
omniroute-cli up
omniroute-cli start [omniroute|openwebui]
omniroute-cli stop [omniroute|openwebui]
omniroute-cli restart [omniroute|openwebui]
omniroute-cli down
omniroute-cli pull
omniroute-cli update
omniroute-cli rebuild
omniroute-cli recreate
omniroute-cli status
omniroute-cli logs [-f] [--tail N] [omniroute|openwebui]
omniroute-cli log [--tail N] [omniroute|openwebui]
omniroute-cli images
omniroute-cli services
omniroute-cli resolved-images
omniroute-cli shell [omniroute|openwebui]
omniroute-cli urls
omniroute-cli config
omniroute-cli doctor
omniroute-cli clean
omniroute-cli prune
omniroute-cli clean-all --yes
omniroute-cli version
omniroute-cli --version
```

`update`/`rebuild` validates the configuration, pulls both images and runs:

```bash
docker compose up -d --force-recreate --remove-orphans
docker compose ps
```

It intentionally does **not** run `docker compose down` first, reducing avoidable downtime and leaving the existing stack running if image pulling fails.

`clean` removes only this Compose stack's containers/network. Global image pruning is available only through the explicit `prune` command.

Persistent volumes are retained except with `clean-all --yes`.

## Global CLI options

The stack definition is `docker-compose.yaml`.

```text
--project-dir DIR      project directory containing docker-compose.yaml
--compose-file FILE    Compose file, default docker-compose.yaml
--dry-run              print Docker commands without executing them
--version, -v          print CLI version
```

Read-only commands such as `status`, `logs`, `images` and `services` do not create or modify `.env`.

## Development

```bash
just fmt
just test
just check
just build
```

`just check` validates Semantic Versioning, formatting, `go vet` and all tests. The project has no third-party Go dependencies.

## Releases

`VERSION` uses Semantic Versioning. `just release` requires a clean Git checkout and creates a reproducible archive from **committed files only**:

```text
dist/omniroute-cli-v<MAJOR.MINOR.PATCH>.zip
```

The ZIP contains a matching top-level directory `omniroute-cli-v<version>/` and never includes local ignored files such as `.env`.

## update-cli integration

`setup.yaml` uses schema 2. Go and Docker are required because secure environment initialization is implemented by `omniroute-cli` itself.

```bash
update-cli --setup
update-cli --setup-workflow update
update-cli --setup-workflow rebuild
update-cli --setup-workflow build-cli
update-cli --setup-workflow check-cli
update-cli --setup-workflow status
```

## CI

GitHub Actions checks every push and pull request for:

- Semantic Version validity
- `gofmt`
- `go vet`
- `go test ./...`
- successful binary build
- CLI help/version smoke tests
