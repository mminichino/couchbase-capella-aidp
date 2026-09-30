data "couchbase-capella-aidp_workflow_runs" "example" {
  organization_id = var.organization_id
  project_id      = var.project_id
  cluster_id      = var.cluster_id
  workflow_id     = var.workflow_id
}
