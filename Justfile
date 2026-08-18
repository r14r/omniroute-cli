set dotenv-load := true

# Show available commands
default:
    @just --list

# Copy .env.example to .env when .env does not exist
init:
    @if [ -f .env ]; then \
        echo ".env already exists"; \
    else \
        cp .env.example .env; \
        echo "Created .env from .env.example"; \
    fi

# Validate the Docker Compose configuration
config: init
    docker compose config

# Pull images and create/start the complete stack
up: init
    docker compose pull
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

# Pull latest service images
pull: init
    docker compose pull

# Pull images and recreate containers; keep persistent volumes
update: init
    docker compose pull
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
      echo "OmniRoute:  http://localhost:$${OMNIROUTE_PUBLIC_PORT:-20128}"; \
      echo "Open WebUI: http://localhost:$${OPENWEBUI_PUBLIC_PORT:-20000}"

# Show resolved Compose service names
services:
    docker compose config --services

# Back up both persistent data volumes
backup: backup-omniroute backup-openwebui

# Back up OmniRoute data to ./backups
backup-omniroute:
    @mkdir -p backups
    docker run --rm \
        -v ai-tools-omniroute-data:/data:ro \
        -v "${PWD}/backups:/backup" \
        alpine:latest \
        sh -c 'tar czf /backup/ai-tools-omniroute-$$(date +%Y%m%d-%H%M%S).tar.gz -C /data .'

# Back up Open WebUI data to ./backups
backup-openwebui:
    @mkdir -p backups
    docker run --rm \
        -v ai-tools-omniroute-openwebui-data:/data:ro \
        -v "${PWD}/backups:/backup" \
        alpine:latest \
        sh -c 'tar czf /backup/ai-tools-omniroute-openwebui-$$(date +%Y%m%d-%H%M%S).tar.gz -C /data .'

# Remove containers/network and unused images; KEEP persistent application data
clean:
    docker compose down --remove-orphans
    docker image prune -f

# Remove containers, network AND persistent volumes. DELETES ALL APPLICATION DATA.
clean-all:
    @printf 'This deletes all ai-tools-omniroute data. Type DELETE to continue: '; \
    read answer; \
    if [ "$${answer}" = "DELETE" ]; then \
        docker compose down -v --remove-orphans; \
    else \
        echo "Cancelled"; \
        exit 1; \
    fi
