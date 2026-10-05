data "identitynow_authorization_right_sets" "identity" {
  category = "identity"
}

locals {
  top_level_identity_right_set_ids = [for r in data.identitynow_authorization_right_sets.identity.right_sets : r.id if r.depth == 0]
}
