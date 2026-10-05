resource "identitynow_dimension" "multivalue" {
  role_id     = "2c91808a7813090a017813b6301fabcd"
  name        = "Multi-Value Dimension"
  description = "A dimension matching multiple job codes"

  owner {
    id   = "2c9180867624cbd7017642d8c8c81f67"
    type = "IDENTITY"
    name = "Example Owner"
  }

  membership {
    type = "STANDARD"

    criteria {
      operation = "EQUALS"
      values    = ["G8244", "G8243", "G8242", "G6644"]

      key {
        type     = "IDENTITY"
        property = "attribute.jobcode"
      }
    }
  }
}
