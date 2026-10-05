resource "identitynow_recommendations_config" "this" {
  recommender_features            = ["jobTitle", "department", "location"]
  peer_group_percentage_threshold = 0.5
}
