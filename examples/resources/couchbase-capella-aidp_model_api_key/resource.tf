resource "couchbase-capella-aidp_model_api_key" "example" {
  organization_id = var.organization_id
  name            = "inference-key"
  description     = "API key for accessing models in the region"
  expiry          = 180
  region          = "us-east-1"
  allowed_cidrs   = ["0.0.0.0/0"]
  allowed_models  = ["*"]
}
