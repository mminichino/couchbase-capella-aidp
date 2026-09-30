resource "couchbase-capella-aidp_workflow" "example" {
  organization_id = var.organization_id
  project_id      = var.project_id
  cluster_id      = var.cluster_id
  name            = "my-vectorization-workflow"
  type            = "vectorization"
  configuration = jsonencode({
    targetCouchbaseKeyspace = {
      bucket     = "my-bucket"
      scope      = "my-scope"
      collection = "my-collection"
    }
    vectorizationConfig = {
      createIndexes = true
      embeddingFieldMappings = {
        vectorEmbeddingField1 = {
          sourceFields = ["field1", "field2"]
        }
      }
      embeddingModel = {
        external = {
          openAiIntegration = {
            providerId = var.openai_provider_id
          }
          modelName = "text-embedding-3-small"
        }
      }
    }
  })
}
