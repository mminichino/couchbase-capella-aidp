package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccAIDPProvider_OpenAI(t *testing.T) {
	requireEnv(t)
	skipIfNoOpenAIKey(t)

	name := randomStringWithPrefix("tf-acc-openai-")
	resourceName := "couchbase-capella-aidp_provider.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAIDPProviderOpenAIConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "organization_id", globalOrgId),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "type", "openAI"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"configuration"},
				ImportStateIdFunc:       importIDOrgResource(resourceName),
			},
			{
				Config: testAccAIDPProviderOpenAIConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
	})
}

func testAccAIDPProviderOpenAIConfig(name string) string {
	return providerConfig() + fmt.Sprintf(`
resource "couchbase-capella-aidp_provider" "test" {
  organization_id = %q
  name            = %q
  type            = "openAI"
  configuration = jsonencode({
    apiKey = %q
  })
}
`, globalOrgId, name, globalOpenAIAPIKey)
}

func importIDOrgResource(resourceName string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceName)
		}
		org := rs.Primary.Attributes["organization_id"]
		id := rs.Primary.ID
		if org == "" || id == "" {
			return "", fmt.Errorf("missing organization_id or id on %s", resourceName)
		}
		return org + "/" + id, nil
	}
}
