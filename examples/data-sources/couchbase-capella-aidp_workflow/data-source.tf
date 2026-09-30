data "couchbase-capella-aidp_workflow" "example" {
  organization_id = var.organization_id
  project_id      = var.project_id
  cluster_id      = var.cluster_id
  id              = var.workflow_id
}
