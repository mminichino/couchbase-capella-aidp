package tests

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccModelAPIKey(t *testing.T) {
	requireEnv(t)
	model := requireExistingModel(t)

	name := randomStringWithPrefix("tf-acc-model-key-")
	resourceName := "couchbase-capella-aidp_model_api_key.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccModelAPIKeyConfig(name, model.Region, model.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "organization_id", globalOrgId),
					resource.TestCheckResourceAttr(resourceName, "name", name),
					resource.TestCheckResourceAttr(resourceName, "region", model.Region),
					resource.TestCheckResourceAttr(resourceName, "expiry", "30"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
					resource.TestCheckResourceAttrSet(resourceName, "token"),
				),
			},
			{
				ResourceName:            resourceName,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"token", "allowed_models", "expiry"},
				ImportStateIdFunc:       importIDOrgResource(resourceName),
			},
		},
	})
}

func TestAccModelAPIKeysDataSource(t *testing.T) {
	requireEnv(t)
	model := requireExistingModel(t)

	name := randomStringWithPrefix("tf-acc-model-keys-ds-")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccModelAPIKeysDataSourceConfig(name, model.Region, model.ID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.couchbase-capella-aidp_model_api_keys.test", "data.#"),
				),
			},
		},
	})
}

func testAccModelAPIKeyConfig(name, region, modelID string) string {
	return providerConfig() + fmt.Sprintf(`
resource "couchbase-capella-aidp_model_api_key" "test" {
  organization_id = %q
  name            = %q
  description     = "acceptance test key"
  expiry          = 30
  region          = %q
  allowed_cidrs   = ["0.0.0.0/0"]
  allowed_models  = [%q]
}
`, globalOrgId, name, region, modelID)
}

func testAccModelAPIKeysDataSourceConfig(name, region, modelID string) string {
	return testAccModelAPIKeyConfig(name, region, modelID) + fmt.Sprintf(`
data "couchbase-capella-aidp_model_api_keys" "test" {
  organization_id = %q
  filter_by       = "region:eq:%s"

  depends_on = [couchbase-capella-aidp_model_api_key.test]
}
`, globalOrgId, region)
}

type existingModel struct {
	ID     string
	Region string
	Status string
}

func requireExistingModel(t *testing.T) existingModel {
	t.Helper()

	url := fmt.Sprintf("%s/v4/organizations/%s/aiServices/models?page=1&perPage=25", globalHost, globalOrgId)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("build list models request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+globalToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("list models: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list models: status %d body %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		Data []struct {
			Model *struct {
				ID          string `json:"id"`
				Status      string `json:"status"`
				CloudConfig *struct {
					Region string `json:"region"`
				} `json:"cloudConfig"`
			} `json:"model"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		t.Fatalf("unmarshal models: %v", err)
	}
	for _, item := range parsed.Data {
		if item.Model == nil || item.Model.ID == "" {
			continue
		}
		region := globalModelRegion
		if item.Model.CloudConfig != nil && item.Model.CloudConfig.Region != "" {
			region = item.Model.CloudConfig.Region
		}
		return existingModel{ID: item.Model.ID, Region: region, Status: item.Model.Status}
	}
	t.Skip("no existing Capella AI models found; create a model before running model API key acceptance tests")
	return existingModel{}
}
