resource "identitynow_ai_access_request_recommendations_config" "this" {
  score_threshold           = 0.5
  restriction_attribute     = "location"
  use_restriction_attribute = true
}
