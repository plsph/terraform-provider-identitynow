resource "identitynow_transform" "country_lookup" {
  name = "Country Lookup"
  type = "lookup"
  attributes_json = jsonencode({
    table = {
      US      = "United States"
      PL      = "Poland"
      default = "Unknown"
    }
  })
}
