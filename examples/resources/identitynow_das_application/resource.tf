resource "identitynow_das_application" "file_share" {
  name             = "Finance File Share"
  description      = "Finance department file share"
  application_type = 8

  application_crawler_settings_json = jsonencode({
    isEnabled = true
    clusterId = "<DAS_CLUSTER_ID>"
  })

  permission_collector_settings_json = jsonencode({
    isEnabled                     = true
    clusterId                     = "<DAS_CLUSTER_ID>"
    calculateEffectivePermissions = true
  })

  data_classification_settings_json = jsonencode({
    isEnabled = false
  })

  activity_configuration_settings_json = jsonencode({
    isEnabled           = true
    clusterId           = "<DAS_CLUSTER_ID>"
    retentionTimePeriod = 90
    retentionTimeType   = "Days"
  })

  tag {
    key   = 1
    value = "finance"
  }
}
