# Terraform Provider for Couchbase Capella AI Data Plane

Terraform provider for managing Couchbase Capella [AI Data Plane](https://docs.couchbase.com/cloud/management-api-reference/index.html#tag/AI-Data-Plane-Providers) resources that are not covered by the official [`couchbase-capella`](https://registry.terraform.io/providers/couchbasecloud/couchbase-capella) provider.

Published under the Terraform Registry namespace **`mminichino`**.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.5
- [Go](https://golang.org/doc/install) >= 1.27 (for building from source)

## Provider Configuration

```hcl
terraform {
  required_providers {
    couchbase-capella-aidp = {
      source  = "mminichino/couchbase-capella-aidp"
      version = "~> 1.0"
    }
  }
}

provider "couchbase-capella-aidp" {
  authentication_token       = var.couchbasecapella_auth_token
  host                       = "https://cloudapi.cloud.couchbase.com" # optional
  global_api_request_timeout = 300                                   # optional, seconds
}
```

### Environment Variables

| Variable | Description |
|---|---|
| `CAPELLA_AUTHENTICATION_TOKEN` | Capella Management API token |
| `CAPELLA_HOST` | API host (default `https://cloudapi.cloud.couchbase.com`) |
| `CAPELLA_GLOBAL_API_REQUEST_TIMEOUT` | Request timeout in seconds (default `300`, minimum `300`) |

## Resources

| Resource | API |
|---|---|
| `couchbase-capella-aidp_provider` | [AI Data Plane Providers](https://docs.couchbase.com/cloud/management-api-reference/index.html#tag/AI-Data-Plane-Providers) |
| `couchbase-capella-aidp_workflow` | [AI Workflows](https://docs.couchbase.com/cloud/management-api-reference/index.html#tag/AI-Workflows) |
| `couchbase-capella-aidp_workflow_run` | AI Workflow runs (start/stop) |
| `couchbase-capella-aidp_model` | [Models (AI Data Plane)](https://docs.couchbase.com/cloud/management-api-reference/index.html#tag/Models-(AI-Data-Plane)) |
| `couchbase-capella-aidp_model_activation` | Model pause/resume |
| `couchbase-capella-aidp_model_api_key` | [Model Services API Keys](https://docs.couchbase.com/cloud/management-api-reference/index.html#tag/Model-Services-API-Keys-(AI-Data-Plane)) |

## Data Sources

| Data Source | Description |
|---|---|
| `couchbase-capella-aidp_provider` / `_providers` | Get / list AIDP providers |
| `couchbase-capella-aidp_workflow` / `_workflows` | Get / list AI workflows |
| `couchbase-capella-aidp_workflow_run` / `_workflow_runs` | Get / list workflow runs |
| `couchbase-capella-aidp_model` / `_models` | Get / list models |
| `couchbase-capella-aidp_model_api_key` / `_model_api_keys` | Get / list model API keys |
| `couchbase-capella-aidp_model_connection_string` | Model inference endpoint |

## Example

```hcl
resource "couchbase-capella-aidp_provider" "openai" {
  organization_id = var.organization_id
  name            = "my-openai"
  type            = "openAI"
  configuration = jsonencode({
    apiKey = var.openai_api_key
  })
}

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

  quantization = "fp16"
  optimization = "throughput"

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

resource "couchbase-capella-aidp_model_api_key" "inference" {
  organization_id = var.organization_id
  name            = "inference-key"
  description     = "API key for model inference"
  expiry          = 180
  region          = "us-east-1"
  allowed_cidrs   = ["0.0.0.0/0"]
  allowed_models  = ["*"]
}
```

## Building Locally

```bash
make build
make install   # installs into ~/.terraform.d/plugins for local development
make generate-docs
```

## Acceptance Tests

Requires a live Capella organization:

```bash
export TF_ACC=1
export CAPELLA_AUTHENTICATION_TOKEN=...
export TF_VAR_organization_id=...
export TF_VAR_openai_api_key=...          # for provider/workflow tests
export TF_VAR_project_id=...              # for workflow tests
export TF_VAR_cluster_id=...
export TF_VAR_model_region=us-east-1
# export ACC_ENABLE_MODEL=true            # optional; deploys a real model (slow/expensive)

make testacc
```

See [tests/README.md](tests/README.md) and [RELEASE.md](RELEASE.md).

## Releasing

```bash
git tag v0.1.0
git push origin v0.1.0
```

GoReleaser + GitHub Actions publish signed release assets for the Terraform Registry. Details in [RELEASE.md](RELEASE.md).

## Import

```bash
terraform import couchbase-capella-aidp_provider.openai <organizationId>/<providerId>
terraform import couchbase-capella-aidp_model.embed <organizationId>/<modelId>
terraform import couchbase-capella-aidp_workflow.vec <organizationId>/<projectId>/<clusterId>/<workflowId>
```

## License

MIT
