## v0.3.0 (2026-09-02)

### Feat

- support the gem plugin, reading Pulp's schema from the running container
- add retain_repo_versions, and make the documented environment variables work

### Refactor

- derive schema, request body and validation from one field table

## v0.2.5 (2026-05-07)

### Fix

- add option to force ipv4 usage

## v0.2.3 (2026-04-29)

### Fix

- add distributions and make namespace readonly
- add namespace to distribution

## v0.2.2 (2026-04-29)

### Fix

- add private field to distribution

## v0.2.1 (2026-04-27)

### Fix

- removing an object role returns 201 not 200
- **client**: differentiate between status not found and no content
- replace pypi with python in validation
- add pulp_href validators
- improve error logging in object_role

## v0.2.0 (2026-04-24)

### Feat

- add generic object role

### Fix

- add validators and list modifiers
- **user**: password may be null

## v0.1.0 (2026-04-22)

### Feat

- add role and user_role
- add user and group resource
- add function to convert data to a string list
- add contentguard resource

### Fix

- ImportState is more flexible and returns parts
- change users and groups in in contentguard to be computed

### Refactor

- move import code to util

## v0.0.2 (2026-04-13)

### Feat

- rename endpoint to server_url

## v0.0.1 (2026-04-10)

### Feat

- initial commit
