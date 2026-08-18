# omniroute-cli

Go CLI and Docker Compose stack for running **OmniRoute** together with **Open WebUI** under the `ai-tools-omniroute` project/service prefix.

## Quick start

```bash
cp .env.example .env
# edit .env if required

just build
./bin/omniroute-cli up
./bin/omniroute-cli status
```

Or install the CLI into `/usr/local/bin`:

```bash
just install
omniroute-cli up
```

The CLI automatically creates `.env` from `.env.example` when it is missing.

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
omniroute-cli clean-all --yes
omniroute-cli version
```

`update` and `rebuild` perform:

1. validate the Compose configuration;
2. pull the OmniRoute image;
3. pull the Open WebUI image;
4. `docker compose down --remove-orphans`;
5. `docker compose up -d --force-recreate --remove-orphans`;
6. show `docker compose ps`.

Persistent Docker volumes are retained. `clean-all --yes` is the explicit destructive command that also removes volumes.

## Global CLI options

```text
--project-dir DIR      project directory containing compose.yaml
--compose-file FILE    Compose file, default compose.yaml
--dry-run              print Docker commands without executing them
```

Example from outside the checkout:

```bash
omniroute-cli --project-dir /opt/omniroute status
```

## Services and ports

| Service | Compose service | Default public port |
|---|---|---:|
| OmniRoute | `ai-tools-omniroute` | 20128 |
| Open WebUI | `ai-tools-omniroute-openwebui` | 20000 |

Public ports are configured in `.env`:

```dotenv
OMNIROUTE_PUBLIC_PORT=20128
OPENWEBUI_PUBLIC_PORT=20000
```

## Development

```bash
just fmt
just test
just check
just build
just cli status
```

The project has no third-party Go dependencies.

## update-cli integration

`setup.yaml` uses schema 2 and provides workflows for Docker lifecycle operations plus optional Go CLI build/check steps:

```bash
update-cli --setup
update-cli --setup-workflow update
update-cli --setup-workflow rebuild
update-cli --setup-workflow build-cli
update-cli --setup-workflow check-cli
update-cli --setup-workflow status
```

Go is optional for Docker deployment. If Go is installed, the setup/update workflow also tests and builds `bin/omniroute-cli`.
