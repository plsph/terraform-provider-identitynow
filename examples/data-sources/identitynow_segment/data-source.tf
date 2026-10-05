data "identitynow_segment" "austin" {
  name = "Austin employees"
}

output "segment_id" {
  value = data.identitynow_segment.austin.id
}
