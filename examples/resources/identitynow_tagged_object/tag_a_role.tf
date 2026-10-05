resource "identitynow_role" "example" {
  name        = "Finance Role"
  description = "Access for the finance department"

  owner {
    id   = "2c9180867624cbd7017642d8c8c81f67"
    type = "IDENTITY"
    name = "John Doe"
  }
}

resource "identitynow_tagged_object" "role_tags" {
  object_type = "ROLE"
  object_ids  = [identitynow_role.example.id]
  tags        = ["critical", "audit-required"]
}
