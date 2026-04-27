resource "pulp_contentguard" "distribution" {
  content_type = "core"
  plugin_name  = "content_redirect"
  name         = "content redirect"
}
