resource "identitynow_network_config" "this" {
  range       = ["10.0.0.0/8", "192.168.1.10"]
  geolocation = ["PL", "DE"]
  whitelisted = true
}
