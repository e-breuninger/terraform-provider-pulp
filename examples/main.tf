terraform {
  required_providers {
    pulp = {
      source = "e-breuninger/pulp"
    }
  }
}

provider "pulp" {
  endpoint = "http://localhost"
  username = "admin"
  password = "gHT9PVdsUZr223vHRKKaTJRigHQJEfF1"
}

resource "pulp_remote" "npm" {
  content_type = "npm"
  plugin_name  = "npm"
  url          = "https://registry.npmjs.org/"
  name         = "bar"
}

resource "pulp_distribution" "npm" {
  content_type = "npm"
  plugin_name  = "npm"
  name         = "baz"
  base_path    = "foo"
  remote       = pulp_remote.npm.pulp_href
}

resource "pulp_remote" "pypi" {
  content_type = "npm"
  plugin_name  = "npm"
  url          = "https://registry.npmjs.org/"
  name         = "bar"
}

resource "pulp_distribution" "python" {
  content_type = "npm"
  plugin_name  = "npm"
  name         = "baz"
  base_path    = "foo"
  remote       = pulp_remote.npm.pulp_href
}
