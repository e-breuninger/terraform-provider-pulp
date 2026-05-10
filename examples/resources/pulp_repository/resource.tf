# Minimal repository — keeps all versions (default)
resource "pulp_repository" "npm" {
  content_type = "npm"
  plugin_name  = "npm"
  name         = "my-npm"
  description  = "NPM mirror"
}

# Repository keeping only the last 10 versions
resource "pulp_repository" "npm_limited" {
  content_type         = "npm"
  plugin_name          = "npm"
  name                 = "my-npm-limited"
  description          = "NPM mirror with version cap"
  retain_repo_versions = 10
}

# File repository with version cap and custom labels
resource "pulp_repository" "files" {
  content_type         = "file"
  plugin_name          = "file"
  name                 = "my-files"
  description          = "Generic file repository"
  retain_repo_versions = 5
  pulp_labels = {
    env  = "production"
    team = "platform"
  }
}
