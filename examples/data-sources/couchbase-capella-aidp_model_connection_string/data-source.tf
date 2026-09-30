data "couchbase-capella-aidp_model_connection_string" "example" {
  organization_id = var.organization_id
  model_id        = var.model_id
}

output "connection_string" {
  value = data.couchbase-capella-aidp_model_connection_string.example.connection_string
}
