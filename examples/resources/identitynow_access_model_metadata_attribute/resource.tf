resource "identitynow_access_model_metadata_attribute" "data_sensitivity" {
  name         = "Data Sensitivity"
  description  = "Sensitivity of the data the access grants"
  type         = "governance"
  object_types = ["all"]
  multiselect  = false

  values {
    value = "public"
    name  = "Public"
  }

  values {
    value = "confidential"
    name  = "Confidential"
  }
}
