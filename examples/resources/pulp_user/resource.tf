resource "pulp_user" "breuninger" {
  username   = "e.breuninger"
  password   = "supersecret123"
  first_name = "E."
  last_name  = "Breuninger"
  email      = "info@breuninger.de"
  is_staff   = true
  is_active  = true
}
