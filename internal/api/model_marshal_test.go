package api

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCloudConfigJSONOmitsRam(t *testing.T) {
	b, err := json.Marshal(CloudConfig{
		Provider: "aws",
		Region:   "us-east-1",
		Compute:  CloudConfigCompute{Cpu: 4, GpuMemory: 24},
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if strings.Contains(s, `"ram"`) {
		t.Fatalf("unexpected ram in payload: %s", s)
	}
	if !strings.Contains(s, `"gpuMemory":24`) {
		t.Fatalf("expected gpuMemory in payload: %s", s)
	}
}
