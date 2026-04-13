# Copyright (c) E. Breuninger GmbH & Co

terraform {
  required_providers {
    pulp = {
      source = "e-breuninger/pulp"
    }
  }
}

provider "pulp" {
  server_url = "http://localhost:8080"
  username   = "admin"
  password   = "admin"
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
