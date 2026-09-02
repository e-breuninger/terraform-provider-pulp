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

# GitHub does not render PlantUML, so diagrams are pre-rendered and committed.
# Sources live in HTML comments in the markdown. See internal/provider/README.md.
diagrams:
	plantuml -headless -tsvg internal/provider/README.md
	@# plantuml omits the trailing newline that end-of-file-fixer wants.
	@printf '\n' >> internal/provider/architecture.svg

# Releasing is manual: nothing in CI bumps the version or writes CHANGELOG.md.
# Pushing the tag is what starts the build.
CZ ?= $(shell command -v cz >/dev/null 2>&1 && echo cz || echo "uv tool run --from commitizen cz")

# cz exits 3 with nothing since the last tag, 21 with nothing releasable.
release-dry-run:
	@out=$$($(CZ) bump --dry-run --yes 2>&1); status=$$?; \
	if [ $$status -eq 3 ] || [ $$status -eq 21 ]; then \
		echo "Nothing since $$(git describe --tags --abbrev=0) warrants a release."; \
		exit 0; \
	fi; \
	echo "$$out"; \
	[ $$status -eq 0 ] || exit $$status; \
	next=$$(echo "$$out" | sed -n "s/^tag to create: //p"); \
	if [ -n "$$next" ] && git rev-parse -q --verify "refs/tags/$$next" >/dev/null; then \
		echo; \
		echo "$$next already exists but is not reachable from here, so this"; \
		echo "release would fail. Delete the stray tag, or name the version"; \
		echo "yourself with: $(CZ) bump <version>"; \
	fi

# Version the Conventional Commits since the last tag, write CHANGELOG.md,
# commit and tag. Needs a clean tree, and the default branch: cz commits the
# changelog and tags that commit, so releasing from a feature branch leaves
# the tag pointing at something master never gets.
RELEASE_BRANCH ?= master

release:
	@branch=$$(git rev-parse --abbrev-ref HEAD); \
	if [ "$$branch" != "$(RELEASE_BRANCH)" ]; then \
		echo "Release from $(RELEASE_BRANCH), not $$branch."; \
		echo "cz tags the commit it writes CHANGELOG.md into, so a tag cut here"; \
		echo "would point at a commit $(RELEASE_BRANCH) never gets."; \
		exit 1; \
	fi; \
	next=$$($(CZ) bump --dry-run --yes 2>/dev/null | sed -n "s/^tag to create: //p"); \
	if [ -n "$$next" ] && git rev-parse -q --verify "refs/tags/$$next" >/dev/null; then \
		echo "$$next already exists, but is not reachable from here."; \
		echo "cz reads the version off the tags it can see, so it would pick a"; \
		echo "name that is already taken. Delete the stray tag, or name the"; \
		echo "version yourself with: $(CZ) bump <version>"; \
		exit 1; \
	fi; \
	status=0; $(CZ) bump || status=$$?; \
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
.PHONY: release release-dry-run changelog diagrams
