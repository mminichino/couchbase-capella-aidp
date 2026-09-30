# Acceptance Tests

These tests exercise the provider against a live Couchbase Capella organization.

## Required environment

| Variable | Required | Purpose |
|---|---|---|
| `TF_ACC` | yes | Must be `1` to enable acceptance tests |
| `CAPELLA_AUTHENTICATION_TOKEN` (or `TF_VAR_auth_token`) | yes | Capella Management API token |
| `TF_VAR_organization_id` | yes | Organization GUID |
| `CAPELLA_HOST` / `TF_VAR_host` | no | Defaults to `https://cloudapi.cloud.couchbase.com` |
| `TF_VAR_openai_api_key` / `OPENAI_API_KEY` | for provider/workflow | Creates an `openAI` AIDP provider |
| `TF_VAR_project_id` / `TF_VAR_cluster_id` | for workflows | Existing project + operational cluster |
| `TF_VAR_model_region` | no | Defaults to `us-east-1` (model API keys) |
| `ACC_ENABLE_MODEL` | no | Set `true` to run model deploy/activation tests (slow; deploys embedding + LLM with caching) |
| `ACC_SKIP_WORKFLOW` | no | Set `true` to skip workflow tests |

## Model deploy tests (`ACC_ENABLE_MODEL`)

`TestAccModel` deploys:

1. Embedding model `nvidia/llama-3.2-nv-embedqa-1b-v2` (cpu=4, gpuMemory=24)
2. LLM `mistralai/mistral-7b-instruct-v0.3` with conversational + semantic caching referencing the embedding model **name**
3. Pause/resume, connection string, and model data sources

Capella returns `202` when a model is queued, then provisions asynchronously. A `deployFailed` status after polling is a Capella-side provisioning failure (capacity, quota, or a stuck model in the org), not a Terraform request-shape error.

If deploys fail repeatedly:

- In Capella UI / API, check for models stuck in `deploying`, `resuming`, or `deployFailed` and destroy them
- Confirm the org is within model quota (`ErrModelLimitExceeded` / code 14034)
- Retry later or in another region via `TF_VAR_model_region` (embedding and LLM must share a region for caching)

## Run

```bash
make testacc
```

Or:

```bash
TF_ACC=1 go test ./tests -v -count=1 -timeout 120m
```

Without `TF_ACC`, the package exits successfully without running tests so `go test ./...` stays green in unit CI.
