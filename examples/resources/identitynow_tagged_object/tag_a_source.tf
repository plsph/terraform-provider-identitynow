data "identitynow_source" "hr" {
  name = "Workday"
}

resource "identitynow_tagged_object" "source_tags" {
  object_type = "SOURCE"
  object_ids  = [data.identitynow_source.hr.id]
  tags        = ["authoritative", "hr-system"]
}
