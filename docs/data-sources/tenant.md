---
subcategory: "Tenant Settings"
page_title: "IdentityNow: Data Source: identitynow_tenant"
description: |-
  Gets information about the current IdentityNow tenant.
---

# Data Source: identitynow_tenant

Use this data source to read information about the current tenant: its name, deployment pod and region, and its products with their licenses.

## Example Usage

```hcl
data "identitynow_tenant" "current" {}

output "tenant_region" {
  value = data.identitynow_tenant.current.region
}

output "tenant_product_names" {
  value = data.identitynow_tenant.current.products[*].product_name
}
```

## Arguments Reference

This data source has no arguments.

## Attributes Reference

* `id` - Tenant ID.
* `name` - Abbreviated tenant name.
* `full_name` - Human-readable tenant name.
* `pod` - Deployment pod of the tenant.
* `region` - Deployment region of the tenant.
* `description` - Tenant description.
* `products` - Products of the tenant. Each product has:
  * `product_name` - Product name.
  * `url` - Product URL.
  * `product_tenant_id` - ID of the product-tenant combination.
  * `product_region` - Product region.
  * `product_right` - Right needed for the product.
  * `api_url` - API URL of the product.
  * `licenses` - Licenses of the product, each with:
    * `license_id` - License name.
    * `legacy_feature_name` - Legacy license name.
  * `attributes_json` - Additional product attributes as a JSON object.
  * `zone` - Zone.
  * `status` - Product status.
  * `status_date_time` - Date of the status.
  * `reason` - Description of the provisioning failure, if any.
  * `notes` - Notes added during tenant provisioning.
  * `date_created` - Creation date of the product.
  * `last_updated` - Last update date of the product.
  * `org_type` - Org type, e.g. `production` or `sandbox`.
