resource "identitynow_password_sync_group" "ad_and_entra" {
  name               = "AD and Entra ID"
  password_policy_id = "2c91808d744ba0ce01746f93b6204199"
  source_ids = [
    "2c9180835d191a86015d28455b4a2329",
    "2c918085744ba0ce01746f93b6204200",
  ]
}
