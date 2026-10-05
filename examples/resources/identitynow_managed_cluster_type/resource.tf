resource "identitynow_managed_cluster_type" "custom" {
  type                = "custom-cluster"
  pod                 = "<POD>"
  org                 = "<ORG>"
  managed_process_ids = ["<MANAGED_PROCESS_ID>"]
}
