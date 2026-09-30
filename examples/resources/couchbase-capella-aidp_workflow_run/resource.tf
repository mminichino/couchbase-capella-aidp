resource "couchbase-capella-aidp_workflow_run" "example" {
  organization_id = var.organization_id
  project_id      = var.project_id
  cluster_id      = var.cluster_id
  workflow_id     = couchbase-capella-aidp_workflow.example.id
}
