resource "couchbase-capella-aidp_model" "embed" {
  organization_id    = var.organization_id
  name               = "my-embedding-model"
  catalog_model_name = "nvidia/llama-3.2-nv-embedqa-1b-v2"

  cloud_config = {
    provider = "aws"
    region   = "us-east-1"
    compute = {
      cpu        = 4
      gpu_memory = 24
    }
  }
}

resource "couchbase-capella-aidp_model" "llm" {
  organization_id    = var.organization_id
  name               = "my-text-generation-model"
  catalog_model_name = "mistralai/mistral-7b-instruct-v0.3"

  cloud_config = {
    provider = "aws"
    region   = "us-east-1"
    compute = {
      cpu        = 4
      gpu_memory = 48
    }
  }

  caching = jsonencode({
    enableStandard       = true
    enableConversational = true
    semantic = {
      embeddingModel = couchbase-capella-aidp_model.embed.name
      scoreThreshold = 0.75
      dimensions     = 2048
      distanceMetric = "dot_product"
    }
    defaultCache = "semantic"
    expiryTTL    = 4000
  })

  depends_on = [couchbase-capella-aidp_model.embed]
}
