package schema

import (
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AIDPProvider is the Terraform model for an AI Data Plane provider integration.
type AIDPProvider struct {
	ID             types.String         `tfsdk:"id"`
	OrganizationID types.String         `tfsdk:"organization_id"`
	Name           types.String         `tfsdk:"name"`
	Type           types.String         `tfsdk:"type"`
	Configuration  jsontypes.Normalized `tfsdk:"configuration"`
	Audit          types.Object         `tfsdk:"audit"`
}

// Workflow is the Terraform model for an AI workflow.
type Workflow struct {
	ID             types.String         `tfsdk:"id"`
	OrganizationID types.String         `tfsdk:"organization_id"`
	ProjectID      types.String         `tfsdk:"project_id"`
	ClusterID      types.String         `tfsdk:"cluster_id"`
	Name           types.String         `tfsdk:"name"`
	Type           types.String         `tfsdk:"type"`
	Configuration  jsontypes.Normalized `tfsdk:"configuration"`
	Audit          types.Object         `tfsdk:"audit"`
}

// WorkflowRun is the Terraform model for an AI workflow run.
type WorkflowRun struct {
	ID             types.String `tfsdk:"id"`
	OrganizationID types.String `tfsdk:"organization_id"`
	ProjectID      types.String `tfsdk:"project_id"`
	ClusterID      types.String `tfsdk:"cluster_id"`
	WorkflowID     types.String `tfsdk:"workflow_id"`
	Status         types.String `tfsdk:"status"`
}

// Model is the Terraform model for a Capella-hosted AI model.
type Model struct {
	ID               types.String         `tfsdk:"id"`
	OrganizationID   types.String         `tfsdk:"organization_id"`
	Name             types.String         `tfsdk:"name"`
	CatalogModelName types.String         `tfsdk:"catalog_model_name"`
	CloudConfig      types.Object         `tfsdk:"cloud_config"`
	Quantization     types.String         `tfsdk:"quantization"`
	Optimization     types.String         `tfsdk:"optimization"`
	Dimensions       types.Int64          `tfsdk:"dimensions"`
	Guardrails       types.List           `tfsdk:"guardrails"`
	Jailbreak        jsontypes.Normalized `tfsdk:"jailbreak"`
	Caching          jsontypes.Normalized `tfsdk:"caching"`
	EnableBatching   types.Bool           `tfsdk:"enable_batching"`
	KeywordFiltering types.List           `tfsdk:"keyword_filtering"`
	Status           types.String         `tfsdk:"status"`
	ConnectionString types.String         `tfsdk:"connection_string"`
	Config           jsontypes.Normalized `tfsdk:"config"`
	Audit            types.Object         `tfsdk:"audit"`
}

// ModelActivation manages pause/resume for a model.
type ModelActivation struct {
	ID              types.String `tfsdk:"id"`
	OrganizationID  types.String `tfsdk:"organization_id"`
	ModelID         types.String `tfsdk:"model_id"`
	ActivationState types.String `tfsdk:"activation_state"`
	Status          types.String `tfsdk:"status"`
}

// ModelAPIKey is the Terraform model for a Model Services API key.
type ModelAPIKey struct {
	ID             types.String  `tfsdk:"id"`
	OrganizationID types.String  `tfsdk:"organization_id"`
	Name           types.String  `tfsdk:"name"`
	Description    types.String  `tfsdk:"description"`
	Expiry         types.Float64 `tfsdk:"expiry"`
	AllowedCIDRs   types.Set     `tfsdk:"allowed_cidrs"`
	AllowedModels  types.List    `tfsdk:"allowed_models"`
	Region         types.String  `tfsdk:"region"`
	Token          types.String  `tfsdk:"token"`
	Audit          types.Object  `tfsdk:"audit"`
}

// Audit schema attribute helpers.
type Audit struct {
	CreatedBy  types.String `tfsdk:"created_by"`
	CreatedAt  types.String `tfsdk:"created_at"`
	ModifiedBy types.String `tfsdk:"modified_by"`
	ModifiedAt types.String `tfsdk:"modified_at"`
	Version    types.Int64  `tfsdk:"version"`
}

// CloudConfigModel maps cloud_config nested attributes.
type CloudConfigModel struct {
	Provider types.String `tfsdk:"provider"`
	Region   types.String `tfsdk:"region"`
	Compute  types.Object `tfsdk:"compute"`
}

// CloudConfigComputeModel maps compute nested attributes.
type CloudConfigComputeModel struct {
	Cpu       types.Int64 `tfsdk:"cpu"`
	GpuMemory types.Int64 `tfsdk:"gpu_memory"`
}
