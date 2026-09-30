data "couchbase-capella-aidp_model_api_keys" "example" {
  organization_id = var.organization_id
  filter_by       = "region:eq:us-east-1"
}
