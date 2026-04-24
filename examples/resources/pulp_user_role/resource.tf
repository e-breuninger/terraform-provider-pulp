resource "pulp_role" "filerepository" {
  name        = "file.filerepository"
  description = "Allow viewing the file repository"
  permissions = [
    "file.view_filerepository",
    "file.delete_filerepository",
  ]
}

resource "pulp_user" "breuninger" {
  username = "breuninger"
  password = "supersecret"
}

resource "pulp_user_role" "breuninger_filerepository" {
  role               = pulp_role.filerepository.name
  user_id            = pulp_user.breuninger.id
  content_object     = null
  content_object_prn = null
}
