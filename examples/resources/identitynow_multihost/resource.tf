variable "sql_server_password" {
  type      = string
  sensitive = true
}

resource "identitynow_multihost" "sql_servers" {
  name        = "SQL Servers"
  description = "Microsoft SQL Server databases"
  connector   = "multihost-microsoft-sql-server"
  connector_attributes_json = jsonencode({
    multiHostAttributes = {
      authType = "SQLAuthentication"
      user     = "svc_identitynow"
      password = var.sql_server_password
    }
  })
  max_sources_per_agg_group = 10
  max_allowed_sources       = 300

  owner {
    id = "2c9180a46faadee4016fb4e018c20639"
  }

  cluster {
    id = "2c9180887de347b4017de8859bf35d3e"
  }
}
