package tests

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
)

func randomStringWithPrefix(prefix string) string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, 8)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))] //nolint:gosec
	}
	return prefix + string(b)
}

func providerConfig() string {
	// Prefer environment credentials (CAPELLA_AUTHENTICATION_TOKEN / CAPELLA_HOST)
	// so acceptance tests exercise the same path as real usage and avoid embedding
	// secrets in generated terraform_plugin_test.tf configs.
	attrs := ""
	if globalHost != "" && globalHost != "https://cloudapi.cloud.couchbase.com" {
		attrs += fmt.Sprintf("  host = %q\n", globalHost)
	}
	return fmt.Sprintf(`
provider "couchbase-capella-aidp" {
%s}
`, attrs)
}

func skipIfNoOpenAIKey(t *testing.T) {
	t.Helper()
	if globalOpenAIAPIKey == "" {
		t.Skip("TF_VAR_openai_api_key / OPENAI_API_KEY not set; skipping AIDP provider acceptance test")
	}
}

func skipIfNoCluster(t *testing.T) {
	t.Helper()
	if globalSkipWorkflow {
		t.Skip("ACC_SKIP_WORKFLOW is set")
	}
	if globalProjectId == "" || globalClusterId == "" {
		t.Skip("TF_VAR_project_id and TF_VAR_cluster_id required for workflow acceptance tests")
	}
}

func skipIfModelDisabled(t *testing.T) {
	t.Helper()
	if !globalEnableModel {
		t.Skip("ACC_ENABLE_MODEL is not set; skipping model deploy acceptance test")
	}
}

func requireEnv(t *testing.T) {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("TF_ACC not set")
	}
}
