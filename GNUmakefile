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

# The version the next release should carry: the increment cz derives from the
# commits, applied to the highest tag in the repository rather than the highest
# one reachable from HEAD.
#
# cz reads the current version off `git describe`, so a tag left behind on a
# branch that was later rebased is invisible to it and it hands out a number
# that is already taken. Taking the highest tag that exists anywhere means a
# release can never reuse a version.
#
# Prints nothing when no commit since the last tag warrants a release.
next-version:
	@increment=$$($(CZ) bump --dry-run --yes 2>/dev/null | sed -n "s/^increment detected: //p"); \
	[ -n "$$increment" ] || exit 0; \
	git tag --list "v[0-9]*" --sort=-v:refname | head -1 | sed "s/^v//" | \
		awk -F. -v inc="$$increment" '{ \
			if (inc == "MAJOR") printf "%d.0.0\n", $$1 + 1; \
			else if (inc == "MINOR") printf "%d.%d.0\n", $$1, $$2 + 1; \
			else printf "%d.%d.%d\n", $$1, $$2, $$3 + 1 }'

# Show the version the next release would carry, and the changelog it would
# write. Changes nothing.
release-dry-run:
	@next=$$($(MAKE) -s next-version); \
	if [ -z "$$next" ]; then \
		echo "Nothing since $$(git describe --tags --abbrev=0) warrants a release."; \
		exit 0; \
	fi; \
	echo "next version: v$$next"; \
	echo; \
	$(CZ) bump --dry-run --yes "$$next"

# Version the commits since the last tag, write CHANGELOG.md, commit and tag.
# Needs a clean tree, and the default branch: cz commits the changelog and tags
# that commit, so a release cut from a feature branch leaves the tag pointing
# at a commit master never gets.
RELEASE_BRANCH ?= master

release:
	@branch=$$(git rev-parse --abbrev-ref HEAD); \
	if [ "$$branch" != "$(RELEASE_BRANCH)" ]; then \
		echo "Release from $(RELEASE_BRANCH), not $$branch."; \
		echo "cz tags the commit it writes CHANGELOG.md into, so a tag cut here"; \
		echo "would point at a commit $(RELEASE_BRANCH) never gets."; \
		exit 1; \
	fi; \
	next=$$($(MAKE) -s next-version); \
	if [ -z "$$next" ]; then \
		echo "Nothing since $$(git describe --tags --abbrev=0) warrants a release."; \
		exit 0; \
	fi; \
	$(CZ) bump --yes "$$next"; \
	echo; \
	echo "Nothing has left this machine yet. To publish:"; \
	echo "    git push --follow-tags"

# Rewrite CHANGELOG.md from history, without bumping or tagging.
changelog:
	$(CZ) changelog

.PHONY: fmt lint test testacc build install generate docker
.PHONY: release release-dry-run next-version changelog diagrams
