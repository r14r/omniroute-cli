# Show available commands
default:
    @just --list

# Show project version
version:
    @cat VERSION

# Create .env when missing and migrate obsolete defaults
init:
    @if [ -f .env ]; then \
        echo ".env already exists"; \
    else \
        cp .env.example .env; \
        chmod 600 .env; \
        echo "Created .env from .env.example"; \
    fi; \
    if grep -q '^OPENWEBUI_IMAGE=openwebui/open-webui:main$' .env; then \
        sed -i.bak 's#^OPENWEBUI_IMAGE=openwebui/open-webui:main$#OPENWEBUI_IMAGE=openwebui/open-webui:latest#' .env; \
        rm -f .env.bak; \
        echo "Migrated Open WebUI image tag: main -> latest"; \
    fi

# Verify Go formatting and run tests
check:
    @files="$(gofmt -l ./cmd ./src)"; \
      if [ -n "$files" ]; then echo "Files require gofmt:"; echo "$files"; exit 1; fi
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

# Validate the Docker Compose configuration
config: init
    docker compose config

# Pull images and create/start the complete stack
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

# Pull images and recreate containers; keep persistent volumes
update: init
    docker compose pull ai-tools-omniroute
    docker compose pull ai-tools-omniroute-openwebui
    docker compose down --remove-orphans
    docker compose up -d --force-recreate --remove-orphans

# Force recreation without pulling images
recreate: init
    docker compose up -d --force-recreate --remove-orphans

# Open a shell in OmniRoute
shell-omniroute:
    docker compose exec ai-tools-omniroute sh

# Open a shell in Open WebUI
shell-openwebui:
    docker compose exec ai-tools-omniroute-openwebui sh

# Print configured public URLs
urls: init
    @. ./.env; \
      echo "OmniRoute:  http://localhost:${OMNIROUTE_PUBLIC_PORT:-20128}"; \
      echo "Open WebUI: http://localhost:${OPENWEBUI_PUBLIC_PORT:-20000}"

# Show resolved Compose service names
services:
    docker compose config --services

# Show the exact images Compose resolves after applying .env
resolved-images: init
    docker compose config --images

# Validate configuration and show resolved services/images
doctor: init
    docker compose config --quiet
    @echo "Services:"
    @docker compose config --services
    @echo "Images:"
    @docker compose config --images

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

# Remove containers/network and unused images; KEEP persistent application data
clean:
    docker compose down --remove-orphans
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

# Create a versioned source ZIP; includes .env.example, excludes runtime data
release: check
    @version="$(cat VERSION)"; \
      mkdir -p dist; \
      archive="dist/omniroute-cli-v$version.zip"; \
      rm -f "$archive"; \
      zip -qr "$archive" . \
        -x '.git/*' '.env' 'bin/*' 'dist/*' 'backups/*' '.DS_Store'; \
      unzip -t "$archive" >/dev/null; \
      echo "Created $archive"
