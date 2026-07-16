data "pulp_remotes" "docker_pull_through" {
  content_type = "container"
  plugin_name  = "pull-through"
}

# The container/pull-through distribution type automatically creates a new
# Remote (and backing Distribution) each time an image is pulled through it.
# This data source lets you discover those dynamically-created Remotes.
output "docker_pull_through_urls" {
  value = [for r in data.pulp_remotes.docker_pull_through.remotes : r.url]
}

# Filters are also supported to narrow down the results.
data "pulp_remotes" "npm_specific" {
  content_type = "npm"
  plugin_name  = "npm"
  name         = "npm"
}
