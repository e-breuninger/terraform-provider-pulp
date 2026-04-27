resource "pulp_remote" "docker" {
  name         = "docker"
  content_type = "container"
  plugin_name  = "pull-through"
  url          = "https://registry-1.docker.io"
}

resource "pulp_remote" "python" {
  name         = "pypi"
  content_type = "python"
  plugin_name  = "python"
  url          = "https://pypi.org/"
}

resource "pulp_remote" "npm" {
  name         = "npm"
  content_type = "npm"
  plugin_name  = "npm"
  url          = "https://registry.npmjs.org"
}

resource "pulp_remote" "maven" {
  name         = "maven"
  content_type = "maven"
  plugin_name  = "maven"
  url          = "https://repo1.maven.org/maven2/"
}
