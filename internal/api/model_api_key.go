package api

// CreateModelAPIKeyRequest creates a Model Services API key.
type CreateModelAPIKeyRequest struct {
	Name          string   `json:"name"`
	Description   *string  `json:"description,omitempty"`
	Expiry        float32  `json:"expiry"`
	AllowedCIDRs  []string `json:"allowedCIDRs"`
	AllowedModels []string `json:"allowedModels,omitempty"`
	Region        string   `json:"region"`
}

// CreateModelAPIKeyResponse is returned on successful API key creation.
type CreateModelAPIKeyResponse struct {
	Id    string `json:"id"`
	Token string `json:"token"`
}

// GetModelAPIKeyResponse is returned when fetching an API key.
type GetModelAPIKeyResponse struct {
	KeyId         string              `json:"keyId"`
	Name          string              `json:"name"`
	Description   *string             `json:"description,omitempty"`
	Expiry        *float32            `json:"expiry,omitempty"`
	AllowedCIDRs  []string            `json:"allowedCIDRs,omitempty"`
	AllowedModels []AllowedModel      `json:"allowedModels,omitempty"`
	Region        string              `json:"region,omitempty"`
	Audit         *CouchbaseAuditData `json:"audit,omitempty"`
}

// AllowedModel is a model reference on an API key.
type AllowedModel struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

// ListModelAPIKeysResponse is returned when listing API keys.
type ListModelAPIKeysResponse struct {
	Data   []GetModelAPIKeyResponse `json:"data"`
	Cursor Cursor                   `json:"cursor"`
}
