# terraform-provider-pulp

The Terraform Pulp provider helps you manage Pulp resources using Terraform. It supports most features provided by Remotes, Repositories, and Distributions for any content type supported by Pulp. The provider is compatible with Pulp 3.x.

See: [Official Documentation](https://registry.terraform.io/providers/e-breuninger/pulp/latest) in the Terraform registry.

Some features may not be included in the provider yet. If you are missing a feature, please open an issue or fork the repository and reference the fork in your issue.

Instead of having different resources for each content type, the provider uses a generic resource for each Pulp resource type.

## Using The Provider

```tf
provider "pulp" {
  server_url = "https://pulp.example.com"
  username   = "admin"
  password   = "<password>"
}

resource "pulp_remote" "docker" {
  name         = "docker"
  content_type = "container"
  plugin_name  = "pull-through"
  url          = "https://registry-1.docker.io"
}

resource "pulp_distribution" "docker" {
  name          = "docker"
  base_path     = "docker"
  content_type  = "container"
  plugin_name   = "pull-through"
  remote        = pulp_remote.docker.pulp_href
}
```

## Building The Provider

1. Clone the repository:

```bash
git clone https://github.com/e-breuninger/terraform-provider-pulp
```

2. Change into the provider directory:

```bash
cd terraform-provider-pulp
```

3. Build the provider:

```bash
go install
```

## Commit Messages And Releases

Commits follow [Conventional Commits](https://www.conventionalcommits.org/).
The version and the changelog are derived from them, so the type matters.

Install the hooks once to catch a malformed message before it is written:

```bash
pre-commit install
```

CI checks pull requests too. A local hook can be skipped, and a fork never ran
it.

Releases are cut deliberately, not on every push. Nothing in CI bumps the
version or writes `CHANGELOG.md`:

```bash
make release-dry-run   # what would the next version be?
make release           # version, CHANGELOG.md, commit and tag
git push --follow-tags # this is what starts the build
```

Run this on `master`, after merging. Commitizen tags the commit it writes
`CHANGELOG.md` into, so a release cut from a feature branch leaves the tag
pointing at a commit `master` never gets.

Pushing the tag starts the `Release` workflow, which builds the provider and
publishes it with release notes grouped by change type.

| commit type                  | effect        |
| ---------------------------- | ------------- |
| `feat:`                      | minor release |
| `fix:`, `perf:`, `refactor:` | patch release |
| `!` or `BREAKING CHANGE:`    | major release |
| everything else              | no release    |

`make release` reports that nothing warrants a release when only types in the
last row have landed. Use `ci:` or `build:` for pipeline and test-harness work.
`fix:` claims a provider bug was fixed, and bumps the version for a change
users never see.

## Contributing

This Provider focuses on the most common use cases for Pulp. We focus on Pull-Through Caches for Maven, PyPI, NPM, and Docker, and may not support all features of Pulp. If you are missing a feature, please open an issue or fork the repository and reference the fork in your issue.

## AI Usage

This project uses AI to assist in code generation and documentation. The AI is used to generate code snippets, documentation, and other content based on the context provided by the developers. The AI is not a replacement for human developers, but rather a tool to assist them in their work.
