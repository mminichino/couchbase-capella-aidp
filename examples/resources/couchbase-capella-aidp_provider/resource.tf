resource "couchbase-capella-aidp_provider" "openai" {
  organization_id = var.organization_id
  name            = "my-openai-provider"
  type            = "openAI"
  configuration = jsonencode({
    apiKey = var.openai_api_key
  })
}

resource "couchbase-capella-aidp_provider" "s3" {
  organization_id = var.organization_id
  name            = "my-s3-provider"
  type            = "awsS3"
  configuration = jsonencode({
    accessKeyId     = var.aws_access_key_id
    secretAccessKey = var.aws_secret_access_key
    awsRegion       = "us-east-1"
    bucket          = "my-aidp-bucket"
    folderPath      = "incoming/"
  })
}
