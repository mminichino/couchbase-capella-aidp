package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccWorkflow_Vectorization(t *testing.T) {
	requireEnv(t)
	skipIfNoCluster(t)
	skipIfNoOpenAIKey(t)

	providerName := randomStringWithPrefix("tf-acc-wf-openai-")
	workflowName := randomStringWithPrefix("tf-acc-wf-")
	resourceName := "couchbase-capella-aidp_workflow.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkflowConfig(providerName, workflowName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "organization_id", globalOrgId),
					resource.TestCheckResourceAttr(resourceName, "project_id", globalProjectId),
					resource.TestCheckResourceAttr(resourceName, "cluster_id", globalClusterId),
					resource.TestCheckResourceAttr(resourceName, "name", workflowName),
					resource.TestCheckResourceAttr(resourceName, "type", "vectorization"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"configuration"},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources[resourceName]
					if !ok {
						return "", fmt.Errorf("not found: %s", resourceName)
					}
					return fmt.Sprintf("%s/%s/%s/%s",
						rs.Primary.Attributes["organization_id"],
						rs.Primary.Attributes["project_id"],
						rs.Primary.Attributes["cluster_id"],
						rs.Primary.ID,
					), nil
				},
			},
		},
	})
}

func TestAccWorkflowRun(t *testing.T) {
	requireEnv(t)
	skipIfNoCluster(t)
	skipIfNoOpenAIKey(t)

	providerName := randomStringWithPrefix("tf-acc-wfr-openai-")
	workflowName := randomStringWithPrefix("tf-acc-wfr-")
	runResource := "couchbase-capella-aidp_workflow_run.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccWorkflowRunConfig(providerName, workflowName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(runResource, "id"),
					resource.TestCheckResourceAttrSet(runResource, "workflow_id"),
				),
			},
		},
	})
}

func testAccWorkflowConfig(providerName, workflowName string) string {
	return testAccAIDPProviderOpenAIConfig(providerName) + fmt.Sprintf(`
resource "couchbase-capella-aidp_workflow" "test" {
  organization_id = %q
  project_id      = %q
  cluster_id      = %q
  name            = %q
  type            = "vectorization"
  configuration = jsonencode({
    targetCouchbaseKeyspace = {
      bucket     = "default"
      scope      = "_default"
      collection = "_default"
    }
    vectorizationConfig = {
      createIndexes = false
      embeddingFieldMappings = {
        vectorEmbeddingField1 = {
          sourceFields = ["text"]
        }
      }
      embeddingModel = {
        external = {
          openAiIntegration = {
            providerId = couchbase-capella-aidp_provider.test.id
          }
          modelName = "text-embedding-3-small"
        }
      }
    }
  })
}
`, globalOrgId, globalProjectId, globalClusterId, workflowName)
}

func testAccWorkflowRunConfig(providerName, workflowName string) string {
	return testAccWorkflowConfig(providerName, workflowName) + `
resource "couchbase-capella-aidp_workflow_run" "test" {
  organization_id = couchbase-capella-aidp_workflow.test.organization_id
  project_id      = couchbase-capella-aidp_workflow.test.project_id
  cluster_id      = couchbase-capella-aidp_workflow.test.cluster_id
  workflow_id     = couchbase-capella-aidp_workflow.test.id
}
`
}
