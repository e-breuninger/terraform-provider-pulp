data "pulp_distribution" "docker" {
  content_type = "container"
  plugin_name  = "pull-through"
  name         = "docker"
  base_path    = "docker"
}

output "docker_distribution_remote" {
  value = data.pulp_distribution.docker.remote
}
