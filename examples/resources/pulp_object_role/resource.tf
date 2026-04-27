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
