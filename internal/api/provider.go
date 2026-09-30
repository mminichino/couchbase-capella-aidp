package api

import "encoding/json"

// CreateProviderRequest creates an AI Data Plane provider integration.
type CreateProviderRequest struct {
	Type          string          `json:"type"`
	Name          string          `json:"name"`
	Configuration json.RawMessage `json:"configuration"`
}

// CreateProviderResponse is returned on successful provider creation.
type CreateProviderResponse struct {
	Id string `json:"id"`
}

// UpdateProviderRequest updates an AI Data Plane provider integration.
type UpdateProviderRequest struct {
	Configuration json.RawMessage `json:"configuration"`
}

// GetProviderResponse is returned when fetching a provider.
type GetProviderResponse struct {
	Id            string              `json:"id,omitempty"`
	Name          *string             `json:"name,omitempty"`
	Type          *string             `json:"type,omitempty"`
	Configuration json.RawMessage     `json:"configuration,omitempty"`
	Audit         *CouchbaseAuditData `json:"audit,omitempty"`
}

// ListProvidersResponse is returned when listing providers.
type ListProvidersResponse struct {
	Data   []GetProviderResponse `json:"data"`
	Cursor Cursor                `json:"cursor"`
}

// ProviderListItem includes id for list responses.
type ProviderListItem struct {
	Id            string              `json:"id"`
	Name          string              `json:"name"`
	Type          string              `json:"type"`
	Configuration json.RawMessage     `json:"configuration,omitempty"`
	Audit         *CouchbaseAuditData `json:"audit,omitempty"`
}

// ListProvidersResponseV2 matches documented list payload with id on each item.
type ListProvidersResponseV2 struct {
	Data   []ProviderListItem `json:"data"`
	Cursor Cursor             `json:"cursor"`
}
