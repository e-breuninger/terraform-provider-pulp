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

resource "pulp_remote" "container" {
  content_type = "container"
  plugin_name  = "pull-through"
  url          = "https://registry-1.docker.io"
  name         = "docker"
}

resource "pulp_distribution" "container" {
  content_type = "container"
  plugin_name  = "pull-through"
  name         = "docker"
  base_path    = "docker"
  remote       = pulp_remote.container.pulp_href
}

import {
  to = pulp_contentguard.container
  id = pulp_distribution.container.content_guard
}

resource "pulp_contentguard" "container" {
  content_type = "core"
  plugin_name  = "content_redirect"
  name         = "content redirect"
}
