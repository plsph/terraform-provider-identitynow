resource "identitynow_source" "active_directory" {
  name             = "Active Directory"
  description      = "The Active Directory connector created by terraform"
  connector        = "active-directory"
  authoritative    = false
  delete_threshold = 10

  owner {
    id   = "2c9180867624cbd7017642d8c8c81f67"
    name = "John Doe"
    type = "IDENTITY"
  }

  cluster {
    id   = "2c9180887671ff8c01767b4671fc7d60"
    name = "Primary Cluster"
    type = "CLUSTER"
  }
}
