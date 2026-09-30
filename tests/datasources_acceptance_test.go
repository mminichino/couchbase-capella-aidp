package tests

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccProvidersDataSource(t *testing.T) {
	requireEnv(t)
	skipIfNoOpenAIKey(t)

	name := randomStringWithPrefix("tf-acc-providers-ds-")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProvidersDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.couchbase-capella-aidp_providers.test", "data.#"),
					resource.TestCheckResourceAttrSet("data.couchbase-capella-aidp_provider.test", "id"),
					resource.TestCheckResourceAttr("data.couchbase-capella-aidp_provider.test", "type", "openAI"),
				),
			},
		},
	})
}

func testAccProvidersDataSourceConfig(name string) string {
	return testAccAIDPProviderOpenAIConfig(name) + fmt.Sprintf(`
data "couchbase-capella-aidp_providers" "test" {
  organization_id = %q
  provider_type   = "openAI"

  depends_on = [couchbase-capella-aidp_provider.test]
}

data "couchbase-capella-aidp_provider" "test" {
  organization_id = %q
  id              = couchbase-capella-aidp_provider.test.id
}
`, globalOrgId, globalOrgId)
}

func TestAccModelsDataSource(t *testing.T) {
	requireEnv(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig() + fmt.Sprintf(`
data "couchbase-capella-aidp_models" "test" {
  organization_id = %q
}
`, globalOrgId),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.couchbase-capella-aidp_models.test", "data.#"),
				),
			},
		},
	})
}
