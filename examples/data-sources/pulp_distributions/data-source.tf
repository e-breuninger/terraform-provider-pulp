data "pulp_distributions" "docker_pull_through" {
  content_type = "container"
  plugin_name  = "pull-through"
}

# The container/pull-through distribution type automatically creates a new
# Distribution (and backing Remote) each time an image is pulled through it.
# This data source lets you discover those dynamically-created Distributions.
output "docker_pull_through_base_paths" {
  value = [for d in data.pulp_distributions.docker_pull_through.distributions : d.base_path]
}

# Filters are also supported to narrow down the results.
data "pulp_distributions" "docker_specific" {
  content_type = "container"
  plugin_name  = "pull-through"
  base_path    = "docker/library/nginx"
}
