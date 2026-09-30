package errors

import "errors"

var (
	ErrMarshallingPayload    = errors.New("error marshalling payload")
	ErrConstructingRequest   = errors.New("error constructing request")
	ErrUnmarshallingResponse = errors.New("error unmarshalling response")
	ErrExecutingRequest      = errors.New("error executing request")
	ErrRatelimit             = errors.New("rate limited by Capella API")
	ErrServiceUnavailable    = errors.New("service unavailable")
	ErrGatewayTimeout        = errors.New("gateway timeout")
	ErrGatewayTimeoutForDDL  = errors.New("gateway timeout for DDL")
	ErrNotAString            = errors.New("payload is not a string")
	ErrUnableToValidateAuth  = errors.New("unable to validate authentication token")
	ErrMissingOrganizationId = errors.New("organization_id is required")
	ErrMissingProjectId      = errors.New("project_id is required")
	ErrMissingClusterId      = errors.New("cluster_id is required")
	ErrMissingId             = errors.New("id is required")
	ErrMissingName           = errors.New("name is required")
	ErrMissingType           = errors.New("type is required")
	ErrMissingConfiguration  = errors.New("configuration is required")
	ErrMissingRegion         = errors.New("region is required")
	ErrMissingModelId        = errors.New("model_id is required")
	ErrMissingWorkflowId     = errors.New("workflow_id is required")
	ErrMissingActivation     = errors.New("activation_state is required")
	ErrInvalidActivation     = errors.New("activation_state must be \"on\" or \"off\"")
	ErrInvalidImportID       = errors.New("invalid import id format")
)
