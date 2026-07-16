data "pulp_remote" "docker" {
  pulp_href = "/pulp/api/v3/remotes/container/pull-through/0195b3c2-1234-7abc-8def-0123456789ab/"
}

output "docker_remote_url" {
  value = data.pulp_remote.docker.url
}
