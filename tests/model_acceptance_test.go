package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccModel(t *testing.T) {
	requireEnv(t)
	skipIfModelDisabled(t)

	embedName := randomStringWithPrefix("tf-acc-embed-")
	llmName := randomStringWithPrefix("tf-acc-llm-")
	embedResource := "couchbase-capella-aidp_model.embed"
	llmResource := "couchbase-capella-aidp_model.llm"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEmbeddingModelConfig(embedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(embedResource, "organization_id", globalOrgId),
					resource.TestCheckResourceAttr(embedResource, "name", embedName),
					resource.TestCheckResourceAttr(embedResource, "catalog_model_name", "nvidia/llama-3.2-nv-embedqa-1b-v2"),
					resource.TestCheckResourceAttrSet(embedResource, "id"),
					resource.TestCheckResourceAttr(embedResource, "status", "healthy"),
				),
			},
			{
				ResourceName:            embedResource,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"catalog_model_name", "quantization", "optimization", "jailbreak", "caching", "guardrails", "keyword_filtering", "enable_batching", "dimensions"},
				ImportStateIdFunc:       importIDOrgResource(embedResource),
			},
			{
				Config: testAccEmbeddingAndLLMWithCachingConfig(embedName, llmName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(llmResource, "organization_id", globalOrgId),
					resource.TestCheckResourceAttr(llmResource, "name", llmName),
					resource.TestCheckResourceAttr(llmResource, "catalog_model_name", "mistralai/mistral-7b-instruct-v0.3"),
					resource.TestCheckResourceAttrSet(llmResource, "id"),
					resource.TestCheckResourceAttr(llmResource, "status", "healthy"),
					resource.TestCheckResourceAttrSet(llmResource, "caching"),
				),
			},
			{
				ResourceName:            llmResource,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"catalog_model_name", "quantization", "optimization", "jailbreak", "caching", "guardrails", "keyword_filtering", "enable_batching", "dimensions"},
				ImportStateIdFunc:       importIDOrgResource(llmResource),
			},
			{
				Config: testAccModelsWithActivation(embedName, llmName, "off"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("couchbase-capella-aidp_model_activation.llm", "activation_state", "off"),
					resource.TestCheckResourceAttrSet("couchbase-capella-aidp_model_activation.llm", "status"),
				),
			},
			{
				Config: testAccModelsWithActivationAndConnectionString(embedName, llmName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("couchbase-capella-aidp_model_activation.llm", "activation_state", "on"),
					resource.TestCheckResourceAttrSet("data.couchbase-capella-aidp_model_connection_string.llm", "connection_string"),
					resource.TestCheckResourceAttrSet("data.couchbase-capella-aidp_model.llm", "id"),
					resource.TestCheckResourceAttrSet("data.couchbase-capella-aidp_model.embed", "id"),
				),
			},
		},
	})
}

func testAccEmbeddingModelConfig(embedName string) string {
	return providerConfig() + fmt.Sprintf(`
resource "couchbase-capella-aidp_model" "embed" {
  organization_id    = %q
  name               = %q
  catalog_model_name = "nvidia/llama-3.2-nv-embedqa-1b-v2"

  cloud_config = {
    provider = "aws"
    region   = %q
    compute = {
      cpu        = 4
      gpu_memory = 24
    }
  }
}
`, globalOrgId, embedName, globalModelRegion)
}

func testAccEmbeddingAndLLMWithCachingConfig(embedName, llmName string) string {
	return testAccEmbeddingModelConfig(embedName) + fmt.Sprintf(`
resource "couchbase-capella-aidp_model" "llm" {
  organization_id    = %q
  name               = %q
  catalog_model_name = "mistralai/mistral-7b-instruct-v0.3"

  cloud_config = {
    provider = "aws"
    region   = %q
    compute = {
      cpu        = 4
      gpu_memory = 48
    }
  }

  quantization = "fp16"
  optimization = "throughput"

  # Conversational + semantic caching backed by the embedding model deployed above.
  # Capella resolves semantic.embeddingModel by model name (not id).
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
`, globalOrgId, llmName, globalModelRegion)
}

func testAccModelsWithActivation(embedName, llmName, activationState string) string {
	return testAccEmbeddingAndLLMWithCachingConfig(embedName, llmName) + fmt.Sprintf(`
resource "couchbase-capella-aidp_model_activation" "llm" {
  organization_id  = couchbase-capella-aidp_model.llm.organization_id
  model_id         = couchbase-capella-aidp_model.llm.id
  activation_state = %q
}
`, activationState)
}

func testAccModelsWithActivationAndConnectionString(embedName, llmName string) string {
	return testAccModelsWithActivation(embedName, llmName, "on") + `
data "couchbase-capella-aidp_model_connection_string" "llm" {
  organization_id = couchbase-capella-aidp_model.llm.organization_id
  model_id        = couchbase-capella-aidp_model.llm.id
}

data "couchbase-capella-aidp_model" "llm" {
  organization_id = couchbase-capella-aidp_model.llm.organization_id
  id              = couchbase-capella-aidp_model.llm.id
}

data "couchbase-capella-aidp_model" "embed" {
  organization_id = couchbase-capella-aidp_model.embed.organization_id
  id              = couchbase-capella-aidp_model.embed.id
}
`
}
