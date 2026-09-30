data "couchbase-capella-aidp_models" "example" {
  organization_id = var.organization_id
  model_status    = "healthy"
  model_kind      = "embedding-generation"
}
