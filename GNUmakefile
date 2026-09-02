.PHONY: default
default: fmt lint install generate

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

# Layer-by-layer pull progress is a wall of noise in a CI log, but worth
# keeping locally for a pull this size. GitHub Actions sets CI.
COMPOSE_UP_FLAGS := $(if $(CI),--quiet-pull)

docker:
	cd docker && docker compose up -d $(COMPOSE_UP_FLAGS)

dockerdown:
	cd docker && docker compose down --volumes

.PHONY: testenv
testenv: dockerdown docker
	sleep 5

test: testenv
	go test -v -cover -timeout=120s -parallel=10 ./...

testacc: testenv
	TF_ACC=1 go test -v -cover -timeout 120m ./...

# Releasing is manual: nothing in CI bumps the version or writes CHANGELOG.md.
# Pushing the tag is what starts the build.
CZ ?= $(shell command -v cz >/dev/null 2>&1 && echo cz || echo "uv tool run --from commitizen cz")

# cz exits 3 with nothing since the last tag, 21 with nothing releasable.
release-dry-run:
	@status=0; $(CZ) bump --dry-run || status=$$?; \
	if [ $$status -eq 3 ] || [ $$status -eq 21 ]; then \
		echo "Nothing since $$(git describe --tags --abbrev=0) warrants a release."; \
		exit 0; \
	fi; \
	exit $$status

# Version the Conventional Commits since the last tag, write CHANGELOG.md,
# commit and tag. Needs a clean tree.
release:
	@status=0; $(CZ) bump || status=$$?; \
	if [ $$status -eq 3 ] || [ $$status -eq 21 ]; then \
		echo "Nothing since $$(git describe --tags --abbrev=0) warrants a release."; \
		exit 0; \
	fi; \
	[ $$status -eq 0 ] || exit $$status; \
	echo; \
	echo "Nothing has left this machine yet. To publish:"; \
	echo "    git push --follow-tags"

# Rewrite CHANGELOG.md from history, without bumping or tagging.
changelog:
	$(CZ) changelog

.PHONY: fmt lint test testacc build install generate docker
.PHONY: release release-dry-run changelog
