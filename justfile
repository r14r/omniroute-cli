# Show available commands
default:
    @just --list

# Show project version
version:
    @cat VERSION

# Validate VERSION as Semantic Versioning
check-version:
    @version="$(cat VERSION)"; \
      echo "$version" | grep -Eq '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$' || { \
        echo "Invalid semantic VERSION: $version" >&2; exit 1; \
      }

# Securely create/protect .env and migrate obsolete non-secret defaults
init:
    @version="$(cat VERSION)"; \
      go run -ldflags "-X main.version=$version" ./cmd/omniroute-cli init

# Verify formatting, vet, tests and semantic version
check: check-version
    @files="$(gofmt -l ./cmd ./src)"; \
      if [ -n "$files" ]; then echo "Files require gofmt:"; echo "$files"; exit 1; fi
    go vet ./...
    go test ./...

# Format Go source
fmt:
    gofmt -w ./cmd ./src

# Run Go tests
test:
    go test ./...

# Build ./bin/omniroute-cli with VERSION embedded
build: check
    @mkdir -p bin
    @version="$(cat VERSION)"; \
      go build -trimpath -ldflags "-s -w -X main.version=$version" \
        -o bin/omniroute-cli ./cmd/omniroute-cli
    @echo "Built bin/omniroute-cli v$(cat VERSION)"

# Run the Go CLI. Example: just cli status
cli *args: build
    ./bin/omniroute-cli {{args}}

# Install CLI; default destination is /usr/local/bin/omniroute-cli
install install_dir="/usr/local/bin": build
    @dir="{{install_dir}}"; \
      mkdir -p "$dir"; \
      cp bin/omniroute-cli "$dir/omniroute-cli"; \
      chmod 755 "$dir/omniroute-cli"; \
      echo "Installed $dir/omniroute-cli"

# Remove CLI from /usr/local/bin or supplied directory
uninstall install_dir="/usr/local/bin":
    @dir="{{install_dir}}"; \
      rm -f "$dir/omniroute-cli"; \
      echo "Removed $dir/omniroute-cli"

# Validate the existing Docker Compose configuration; does not create .env
config:
    docker compose config

# Pull images and create/start the complete stack; creates .env if missing
up: init
    docker compose pull ai-tools-omniroute
    docker compose pull ai-tools-omniroute-openwebui
    docker compose up -d --remove-orphans

# Alias for up
run: up

# Start already-created containers
start:
    docker compose start

# Stop containers without removing them
stop:
    docker compose stop

# Restart all services
restart:
    docker compose restart

# Stop/remove containers and network; keep persistent volumes
down:
    docker compose down --remove-orphans

# Show service/container status
status:
    docker compose ps

# Show images used by the stack
images:
    docker compose images

# Show processes inside the containers
top:
    docker compose top

# Follow logs. Example: just logs ai-tools-omniroute
logs service="":
    @if [ -n "{{service}}" ]; then \
        docker compose logs --tail=200 -f "{{service}}"; \
    else \
        docker compose logs --tail=200 -f; \
    fi

# Show the last 200 log lines
log service="":
    @if [ -n "{{service}}" ]; then \
        docker compose logs --tail=200 "{{service}}"; \
    else \
        docker compose logs --tail=200; \
    fi

# Pull latest service images separately for clearer errors
pull: init
    docker compose pull ai-tools-omniroute
    docker compose pull ai-tools-omniroute-openwebui

# Pull images and recreate containers without an explicit down; keep persistent volumes
update: init
    docker compose pull ai-tools-omniroute
    docker compose pull ai-tools-omniroute-openwebui
    docker compose up -d --force-recreate --remove-orphans
    docker compose ps

# Force recreation without pulling images
recreate: init
    docker compose up -d --force-recreate --remove-orphans

# Open a shell in OmniRoute
shell-omniroute:
    docker compose exec ai-tools-omniroute sh

# Open a shell in Open WebUI
shell-openwebui:
    docker compose exec ai-tools-omniroute-openwebui sh

# Print configured public URLs without creating .env
urls:
    @if [ -f .env ]; then . ./.env; fi; \
      echo "OmniRoute:  http://localhost:${OMNIROUTE_PUBLIC_PORT:-20128}"; \
      echo "Open WebUI: http://localhost:${OPENWEBUI_PUBLIC_PORT:-20000}"

# Show resolved Compose service names
services:
    docker compose config --services

# Show the exact images Compose resolves after applying .env
resolved-images:
    docker compose config --images

# Build CLI and run security/config diagnostics
doctor: build
    ./bin/omniroute-cli doctor

# Back up both persistent data volumes
backup: backup-omniroute backup-openwebui

# Back up OmniRoute data to ./backups
backup-omniroute:
    @mkdir -p backups
    docker run --rm \
        -v ai-tools-omniroute-data:/data:ro \
        -v "${PWD}/backups:/backup" \
        alpine:latest \
        sh -c 'tar czf /backup/ai-tools-omniroute-$(date +%Y%m%d-%H%M%S).tar.gz -C /data .'

# Back up Open WebUI data to ./backups
backup-openwebui:
    @mkdir -p backups
    docker run --rm \
        -v ai-tools-omniroute-openwebui-data:/data:ro \
        -v "${PWD}/backups:/backup" \
        alpine:latest \
        sh -c 'tar czf /backup/ai-tools-omniroute-openwebui-$(date +%Y%m%d-%H%M%S).tar.gz -C /data .'

# Remove containers/network only; KEEP volumes and unrelated Docker images
clean:
    docker compose down --remove-orphans

# Explicit global Docker image prune
prune:
    docker image prune -f

# Remove containers, network AND persistent volumes. DELETES ALL APPLICATION DATA.
clean-all:
    @printf 'This deletes all ai-tools-omniroute data. Type DELETE to continue: '; \
    read answer; \
    if [ "$answer" = "DELETE" ]; then \
        docker compose down -v --remove-orphans; \
    else \
        echo "Cancelled"; \
        exit 1; \
    fi

# Create a reproducible ZIP from committed Git files only
release: check
    @git rev-parse --is-inside-work-tree >/dev/null 2>&1 || { echo "release requires a Git checkout" >&2; exit 1; }
    @git diff --quiet && git diff --cached --quiet || { echo "Commit tracked changes before creating a release" >&2; exit 1; }
    @version="$(cat VERSION)"; \
      mkdir -p dist; \
      archive="dist/omniroute-cli-v$version.zip"; \
      rm -f "$archive"; \
      git archive --format=zip --prefix="omniroute-cli-v$version/" --output="$archive" HEAD; \
      unzip -t "$archive" >/dev/null; \
      echo "Created $archive"
