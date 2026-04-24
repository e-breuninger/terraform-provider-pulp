# The import ID consists of two parts joined by `|`:
#   1. the full pulp_href of the target object
#   2. the name of the role
terraform import pulp_object_role.example '/pulp/api/v3/repositories/file/file/ebebebeb-ebeb-ebeb-ebeb-123456789abc/|file.repository_owner'
