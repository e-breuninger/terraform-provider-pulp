resource "pulp_user" "anon" {
  username = "anon"
  password = "ebeb1881!"
}

resource "pulp_remote" "docker" {
  name         = "docker"
  content_type = "container"
  plugin_name  = "pull-through"
  url          = "https://registry-1.docker.io"
}

resource "pulp_distribution" "docker" {
  name         = "docker"
  base_path    = "docker"
  content_type = "container"
  plugin_name  = "pull-through"
  remote       = pulp_remote.docker.pulp_href
  private      = false
}

resource "pulp_object_role" "anon_pullthrough" {
  pulp_href = pulp_distribution.docker.pulp_href
  role      = "container.containerpullthroughdistribution_consumer"
  users     = [pulp_user.anon.username]
}

resource "pulp_object_role" "anon_namespace" {
  pulp_href = pulp_distribution.docker.namespace
  role      = "container.containernamespace_consumer"
  users     = [pulp_user.anon.username]
}
