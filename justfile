default:
    @just --list

version:
    @go run ./cmd/semver-check "$(cat VERSION)"

check-version:
    @go run ./cmd/semver-check "$(cat VERSION)" >/dev/null

fmt:
    gofmt -w ./cmd ./src

test:
    go test ./...

check: check-version
    @files="$(gofmt -l ./cmd ./src)"; test -z "$files" || { echo "Files require gofmt:"; echo "$files"; exit 1; }
    go vet ./...
    go test -race ./...

build: check
    @mkdir -p bin
    @rm -f bin/omniroute-cli
    @version="$(cat VERSION)"; \
      go build -trimpath -ldflags "-s -w -X main.version=$version" -o bin/omniroute-cli ./cmd/omniroute-cli; \
      actual="$(./bin/omniroute-cli --version)"; \
      expected="omniroute-cli v$version"; \
      test "$actual" = "$expected" || { echo "Version mismatch: $actual != $expected" >&2; exit 1; }; \
      echo "Built bin/omniroute-cli v$version"

install install_dir="/usr/local/bin": build
    @dir="{{install_dir}}"; target="$dir/omniroute-cli"; \
      if [ ! -d "$dir" ]; then mkdir -p "$dir" 2>/dev/null || sudo mkdir -p "$dir"; fi; \
      if [ -w "$dir" ]; then install -m 0755 bin/omniroute-cli "$target"; else sudo install -m 0755 bin/omniroute-cli "$target"; fi; \
      "$target" --version

uninstall install_dir="/usr/local/bin":
    @target="{{install_dir}}/omniroute-cli"; \
      if [ ! -e "$target" ]; then echo "$target is not installed"; \
      elif [ -w "$target" ] || [ -w "{{install_dir}}" ]; then rm -f "$target"; \
      else sudo rm -f "$target"; fi

cli *args: build
    ./bin/omniroute-cli {{args}}

# Runtime commands delegate to the Go CLI so lifecycle semantics have one source of truth.
init: build
    ./bin/omniroute-cli init
up: build
    ./bin/omniroute-cli up
run: up
pull: build
    ./bin/omniroute-cli pull
update: build
    ./bin/omniroute-cli update
update-plan: build
    ./bin/omniroute-cli update --plan
rollback: build
    ./bin/omniroute-cli rollback --previous
recreate: build
    ./bin/omniroute-cli recreate
start: build
    ./bin/omniroute-cli start
stop: build
    ./bin/omniroute-cli stop
restart: build
    ./bin/omniroute-cli restart
down: build
    ./bin/omniroute-cli down
status: build
    ./bin/omniroute-cli status
health: build
    ./bin/omniroute-cli health
health-deep: build
    ./bin/omniroute-cli health --deep
doctor: build
    ./bin/omniroute-cli doctor --deep
config: build
    ./bin/omniroute-cli config validate
images: build
    ./bin/omniroute-cli images
top: build
    ./bin/omniroute-cli top
logs: build
    ./bin/omniroute-cli logs
log: build
    ./bin/omniroute-cli log
urls: build
    ./bin/omniroute-cli urls
clean: build
    ./bin/omniroute-cli clean
prune: build
    ./bin/omniroute-cli prune
clean-all: build
    ./bin/omniroute-cli clean-all --yes

# Existing volume backup helpers are retained as developer conveniences only.
# Backup/restore is intentionally not part of omniroute-cli.
backup: backup-omniroute backup-openwebui
backup-omniroute:
    @mkdir -p backups
    docker run --rm -v ai-tools-omniroute-data:/data:ro -v "${PWD}/backups:/backup" alpine:latest sh -c 'tar czf /backup/ai-tools-omniroute-$(date +%Y%m%d-%H%M%S).tar.gz -C /data .'
backup-openwebui:
    @mkdir -p backups
    docker run --rm -v ai-tools-omniroute-openwebui-data:/data:ro -v "${PWD}/backups:/backup" alpine:latest sh -c 'tar czf /backup/ai-tools-omniroute-openwebui-$(date +%Y%m%d-%H%M%S).tar.gz -C /data .'

release: check
    @git rev-parse --is-inside-work-tree >/dev/null 2>&1 || { echo "release requires Git" >&2; exit 1; }
    @git diff --quiet && git diff --cached --quiet || { echo "Commit tracked changes before release" >&2; exit 1; }
    @version="$(go run ./cmd/semver-check "$(cat VERSION)")"; \
      mkdir -p dist; archive="dist/omniroute-cli-v$version.zip"; rm -f "$archive"; \
      git archive --format=zip --prefix="omniroute-cli-v$version/" --output="$archive" HEAD; \
      unzip -t "$archive" >/dev/null; shasum -a 256 "$archive"
