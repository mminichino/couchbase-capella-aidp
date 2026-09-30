package api

import "encoding/json"

// CreateModelRequest creates a Capella-hosted model deployment.
type CreateModelRequest struct {
	Name             string          `json:"name"`
	CatalogModelName string          `json:"catalogModelName"`
	CloudConfig      CloudConfig     `json:"cloudConfig"`
	Quantization     *string         `json:"quantization,omitempty"`
	Optimization     *string         `json:"optimization,omitempty"`
	Dimensions       *int            `json:"dimensions,omitempty"`
	Guardrails       []string        `json:"guardrails,omitempty"`
	Jailbreak        json.RawMessage `json:"jailbreak,omitempty"`
	Caching          json.RawMessage `json:"caching,omitempty"`
	EnableBatching   *bool           `json:"enableBatching,omitempty"`
	KeywordFiltering []string        `json:"keywordFiltering,omitempty"`
}

// CloudConfig is the cloud placement for a model.
type CloudConfig struct {
	Provider string             `json:"provider"`
	Region   string             `json:"region"`
	Compute  CloudConfigCompute `json:"compute"`
}

// CloudConfigCompute describes model compute sizing.
// Capella accepts only cpu and gpuMemory (GB); ram is not a valid create field.
type CloudConfigCompute struct {
	Cpu       int `json:"cpu"`
	GpuMemory int `json:"gpuMemory,omitempty"`
}

// CreateModelResponse is returned on successful model creation.
type CreateModelResponse struct {
	Id string `json:"id"`
}

// UpdateModelRequest updates mutable model settings.
type UpdateModelRequest struct {
	Name             *string         `json:"name,omitempty"`
	Guardrails       []string        `json:"guardrails,omitempty"`
	Jailbreak        json.RawMessage `json:"jailbreak,omitempty"`
	Caching          json.RawMessage `json:"caching,omitempty"`
	EnableBatching   *bool           `json:"enableBatching,omitempty"`
	KeywordFiltering []string        `json:"keywordFiltering,omitempty"`
}

// GetModelResponse wraps a model payload.
type GetModelResponse struct {
	Model *ModelDetails `json:"model"`
}

// ModelDetails is the model resource representation.
type ModelDetails struct {
	Id               string              `json:"id"`
	Name             string              `json:"name"`
	Status           string              `json:"status,omitempty"`
	ConnectionString *string             `json:"connectionString,omitempty"`
	CloudConfig      *CloudConfig        `json:"cloudConfig,omitempty"`
	Config           json.RawMessage     `json:"config,omitempty"`
	UsageMetrics     json.RawMessage     `json:"usageMetrics,omitempty"`
	Actions          []string            `json:"actions,omitempty"`
	Audit            *CouchbaseAuditData `json:"audit,omitempty"`
}

// ListModelsResponse is returned when listing models.
type ListModelsResponse struct {
	Data   []GetModelResponse `json:"data"`
	Cursor Cursor             `json:"cursor"`
}

// GetConnectionStringResponse is returned for model connection string.
type GetConnectionStringResponse struct {
	ConnectionString *string `json:"connectionString,omitempty"`
}
