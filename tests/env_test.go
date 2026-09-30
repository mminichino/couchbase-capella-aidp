package tests

import (
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/mminichino/couchbase-capella-aidp/internal/provider"
)

var (
	globalHost               string
	globalToken              string
	globalOrgId              string
	globalProjectId          string
	globalClusterId          string
	globalOpenAIAPIKey       string
	globalModelRegion        string
	globalEnableModel        bool
	globalSkipWorkflow       bool
	protoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"couchbase-capella-aidp": providerserver.NewProtocol6WithError(provider.New()()),
	}
)

func TestMain(m *testing.M) {
	// Allow unit-style runs of this package without Capella credentials when TF_ACC is unset.
	if os.Getenv("TF_ACC") == "" {
		os.Exit(0)
	}

	if err := loadEnv(); err != nil {
		_, _ = os.Stderr.WriteString("acceptance tests: " + err.Error() + "\n")
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func loadEnv() error {
	globalHost = firstNonEmpty(os.Getenv("CAPELLA_HOST"), os.Getenv("TF_VAR_host"), "https://cloudapi.cloud.couchbase.com")
	globalHost = strings.Trim(strings.TrimSpace(globalHost), `"'`)
	globalToken = firstNonEmpty(os.Getenv("CAPELLA_AUTHENTICATION_TOKEN"), os.Getenv("TF_VAR_auth_token"))
	globalToken = strings.Trim(strings.TrimSpace(globalToken), `"'`)
	if globalToken == "" {
		return errMissing("CAPELLA_AUTHENTICATION_TOKEN or TF_VAR_auth_token")
	}

	globalOrgId = firstNonEmpty(os.Getenv("TF_VAR_organization_id"), os.Getenv("CAPELLA_ORGANIZATION_ID"))
	globalOrgId = strings.Trim(strings.TrimSpace(globalOrgId), `"'`)
	if globalOrgId == "" {
		return errMissing("TF_VAR_organization_id or CAPELLA_ORGANIZATION_ID")
	}

	globalProjectId = strings.Trim(strings.TrimSpace(firstNonEmpty(os.Getenv("TF_VAR_project_id"), os.Getenv("CAPELLA_PROJECT_ID"))), `"'`)
	globalClusterId = strings.Trim(strings.TrimSpace(firstNonEmpty(os.Getenv("TF_VAR_cluster_id"), os.Getenv("CAPELLA_CLUSTER_ID"))), `"'`)
	globalOpenAIAPIKey = strings.Trim(strings.TrimSpace(firstNonEmpty(os.Getenv("TF_VAR_openai_api_key"), os.Getenv("OPENAI_API_KEY"))), `"'`)
	globalModelRegion = strings.Trim(strings.TrimSpace(firstNonEmpty(os.Getenv("TF_VAR_model_region"), os.Getenv("CAPELLA_MODEL_REGION"), "us-east-1")), `"'`)
	globalEnableModel = isTruthy(os.Getenv("ACC_ENABLE_MODEL"))
	globalSkipWorkflow = isTruthy(os.Getenv("ACC_SKIP_WORKFLOW"))

	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func isTruthy(v string) bool {
	switch v {
	case "1", "true", "TRUE", "True", "yes", "YES", "Yes":
		return true
	default:
		return false
	}
}

type missingEnvError string

func (e missingEnvError) Error() string {
	return "missing required environment variable: " + string(e)
}

func errMissing(name string) error {
	return missingEnvError(name)
}
