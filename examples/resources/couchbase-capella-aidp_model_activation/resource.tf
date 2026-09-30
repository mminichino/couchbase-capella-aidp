resource "couchbase-capella-aidp_model_activation" "example" {
  organization_id  = var.organization_id
  model_id         = couchbase-capella-aidp_model.example.id
  activation_state = "on"
}
