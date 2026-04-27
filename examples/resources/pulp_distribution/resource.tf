resource "pulp_contentguard" "distribution" {
  content_type = "core"
  plugin_name  = "content_redirect"
  name         = "content redirect"
}

resource "pulp_object_role" "content_redirect_contentguard_owner" {
  pulp_href = pulp_contentguard.distribution.pulp_href
  role      = "core.contentredirectcontentguard_owner"
  users     = ["admin"]
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
  content_guard = pulp_contentguard.distribution.pulp_href
}
