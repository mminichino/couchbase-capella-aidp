package api

import "encoding/json"

// CreateWorkflowRequest creates an AI workflow.
type CreateWorkflowRequest struct {
	Name          string          `json:"name"`
	Type          string          `json:"type"`
	Configuration json.RawMessage `json:"configuration"`
}

// CreateWorkflowResponse is returned on successful workflow creation.
type CreateWorkflowResponse struct {
	Id string `json:"id"`
}

// GetWorkflowResponse is returned when fetching a workflow.
type GetWorkflowResponse struct {
	Id            string              `json:"id"`
	Name          string              `json:"name"`
	Type          string              `json:"type"`
	Configuration json.RawMessage     `json:"configuration,omitempty"`
	Audit         *CouchbaseAuditData `json:"audit,omitempty"`
}

// ListWorkflowsResponse is returned when listing workflows.
type ListWorkflowsResponse struct {
	Data   []GetWorkflowResponse `json:"data"`
	Cursor Cursor                `json:"cursor"`
}

// CreateWorkflowRunResponse is returned when starting a workflow run.
type CreateWorkflowRunResponse struct {
	Id string `json:"id"`
}

// GetWorkflowRunResponse is returned when fetching a workflow run.
type GetWorkflowRunResponse struct {
	Id     string              `json:"id"`
	Status string              `json:"status,omitempty"`
	Audit  *CouchbaseAuditData `json:"audit,omitempty"`
}

// ListWorkflowRunsResponse is returned when listing workflow runs.
type ListWorkflowRunsResponse struct {
	Data   []GetWorkflowRunResponse `json:"data"`
	Cursor Cursor                   `json:"cursor"`
}

// GetWorkflowRunProcessedFilesResponse is returned when listing processed files.
type GetWorkflowRunProcessedFilesResponse struct {
	Data   json.RawMessage `json:"data"`
	Cursor Cursor          `json:"cursor"`
}
