data "identitynow_tenant" "current" {}

output "tenant_region" {
  value = data.identitynow_tenant.current.region
}

output "tenant_product_names" {
  value = data.identitynow_tenant.current.products[*].product_name
}
