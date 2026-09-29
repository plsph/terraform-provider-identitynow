resource "identitynow_source" "aws_iam_source" {
  name             = "AWS IAM Source"
  description      = "The AWS IAM connector created by terraform"
  connector        = "aws"
  authoritative    = false
  delete_threshold = 10

  owner {
    id   = data.identitynow_identity.john_doe.id
    name = data.identitynow_identity.john_doe.name
    type = "IDENTITY"
  }
}
