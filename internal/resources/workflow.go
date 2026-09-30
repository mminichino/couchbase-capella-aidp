package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mminichino/couchbase-capella-aidp/internal/api"
	"github.com/mminichino/couchbase-capella-aidp/internal/errors"
	providerschema "github.com/mminichino/couchbase-capella-aidp/internal/schema"
)

var (
	_ resource.Resource                = &Workflow{}
	_ resource.ResourceWithConfigure   = &Workflow{}
	_ resource.ResourceWithImportState = &Workflow{}
)

const errorMessageWhileWorkflowCreation = "There is an error during workflow creation. Please check in Capella to see if any hanging resources" +
	" have been created, unexpected error: "

const errorMessageAfterWorkflowCreation = "Workflow creation is successful, but encountered an error while checking the current" +
	" state of the workflow. Please run `terraform plan` after 1-2 minutes to know the" +
	" current workflow state. Additionally, run `terraform apply --refresh-only` to update" +
	" the state from remote, unexpected error: "

// Workflow is the AI Data Plane workflow resource implementation.
type Workflow struct {
	*providerschema.Data
}

// NewWorkflow is a helper function to simplify the provider implementation.
func NewWorkflow() resource.Resource {
	return &Workflow{}
}

func (r *Workflow) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflow"
}

func (r *Workflow) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = WorkflowSchema()
}

func (r *Workflow) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*providerschema.Data)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *providerschema.Data, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.Data = data
}

func (r *Workflow) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan providerschema.Workflow
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateCreateWorkflow(plan); err != nil {
		resp.Diagnostics.AddError(
			"Error creating workflow",
			"Could not create workflow, unexpected error: "+err.Error(),
		)
		return
	}

	createReq := api.CreateWorkflowRequest{
		Name:          plan.Name.ValueString(),
		Type:          plan.Type.ValueString(),
		Configuration: json.RawMessage(plan.Configuration.ValueString()),
	}

	url := workflowCollectionURL(r.HostURL, plan.OrganizationID.ValueString(), plan.ProjectID.ValueString(), plan.ClusterID.ValueString())
	cfg := api.EndpointCfg{Url: url, Method: http.MethodPost, SuccessStatus: http.StatusCreated}
	response, err := r.Client.ExecuteWithRetry(ctx, cfg, createReq, r.Token, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating workflow",
			errorMessageWhileWorkflowCreation+api.ParseError(err),
		)
		return
	}

	var createResp api.CreateWorkflowResponse
	if err := json.Unmarshal(response.Body, &createResp); err != nil {
		resp.Diagnostics.AddError(
			"Error creating workflow",
			errorMessageWhileWorkflowCreation+"error during unmarshalling: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, initializeWorkflowWithPlanAndID(plan, createResp.Id))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	refreshed, err := r.refreshWorkflow(
		ctx,
		plan.OrganizationID.ValueString(),
		plan.ProjectID.ValueString(),
		plan.ClusterID.ValueString(),
		createResp.Id,
	)
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Error reading Capella workflow",
			errorMessageAfterWorkflowCreation+api.ParseError(err),
		)
		return
	}

	diags = resp.State.Set(ctx, refreshed)
	resp.Diagnostics.Append(diags...)
}

func (r *Workflow) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state providerschema.Workflow
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateWorkflowIDs(state); err != nil {
		resp.Diagnostics.AddError(
			"Error reading Capella workflow",
			"Could not read Capella workflow: "+err.Error(),
		)
		return
	}

	refreshed, err := r.refreshWorkflow(
		ctx,
		state.OrganizationID.ValueString(),
		state.ProjectID.ValueString(),
		state.ClusterID.ValueString(),
		state.ID.ValueString(),
	)
	if err != nil {
		resourceNotFound, errString := api.CheckResourceNotFoundError(err)
		if resourceNotFound {
			tflog.Info(ctx, "resource doesn't exist in remote server removing resource from state file")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading Capella workflow",
			"Could not read Capella workflow "+state.ID.ValueString()+": "+errString,
		)
		return
	}

	diags = resp.State.Set(ctx, refreshed)
	resp.Diagnostics.Append(diags...)
}

func (r *Workflow) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// Capella does not support updating workflows. All attributes use RequiresReplace.
}

func (r *Workflow) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state providerschema.Workflow
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateWorkflowIDs(state); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting Capella workflow",
			"Could not delete Capella workflow: "+err.Error(),
		)
		return
	}

	url := workflowURL(
		r.HostURL,
		state.OrganizationID.ValueString(),
		state.ProjectID.ValueString(),
		state.ClusterID.ValueString(),
		state.ID.ValueString(),
	)
	cfg := api.EndpointCfg{Url: url, Method: http.MethodDelete, SuccessStatus: http.StatusAccepted}
	_, err := executeAllowingStatuses(ctx, r.Client, cfg, nil, r.Token, nil, http.StatusNoContent)
	if err != nil {
		resourceNotFound, errString := api.CheckResourceNotFoundError(err)
		if resourceNotFound {
			tflog.Info(ctx, "resource doesn't exist in remote server removing resource from state file")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting Capella workflow",
			"Could not delete Capella workflow "+state.ID.ValueString()+": "+errString,
		)
		return
	}
}

func (r *Workflow) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerschema.SplitImportID(req.ID, 4)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: organizationId/projectId/clusterId/workflowId. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[3])...)
}

func (r *Workflow) refreshWorkflow(
	ctx context.Context, organizationID, projectID, clusterID, workflowID string,
) (*providerschema.Workflow, error) {
	url := workflowURL(r.HostURL, organizationID, projectID, clusterID, workflowID)
	cfg := api.EndpointCfg{Url: url, Method: http.MethodGet, SuccessStatus: http.StatusOK}
	response, err := r.Client.ExecuteWithRetry(ctx, cfg, nil, r.Token, nil)
	if err != nil {
		return nil, err
	}

	var getResp api.GetWorkflowResponse
	if err := json.Unmarshal(response.Body, &getResp); err != nil {
		return nil, fmt.Errorf("error unmarshalling workflow response: %w", err)
	}

	id := getResp.Id
	if id == "" {
		id = workflowID
	}

	configuration := jsontypes.NewNormalizedNull()
	if len(getResp.Configuration) > 0 {
		configuration = jsontypes.NewNormalizedValue(string(getResp.Configuration))
	}

	return &providerschema.Workflow{
		ID:             types.StringValue(id),
		OrganizationID: types.StringValue(organizationID),
		ProjectID:      types.StringValue(projectID),
		ClusterID:      types.StringValue(clusterID),
		Name:           types.StringValue(getResp.Name),
		Type:           types.StringValue(getResp.Type),
		Configuration:  configuration,
		Audit:          providerschema.NewAuditObject(ctx, getResp.Audit),
	}, nil
}

func initializeWorkflowWithPlanAndID(plan providerschema.Workflow, id string) providerschema.Workflow {
	plan.ID = types.StringValue(id)
	plan.Audit = types.ObjectNull(providerschema.AuditAttrTypes())
	return plan
}

func validateCreateWorkflow(plan providerschema.Workflow) error {
	if plan.OrganizationID.IsNull() || plan.OrganizationID.ValueString() == "" {
		return errors.ErrMissingOrganizationId
	}
	if plan.ProjectID.IsNull() || plan.ProjectID.ValueString() == "" {
		return errors.ErrMissingProjectId
	}
	if plan.ClusterID.IsNull() || plan.ClusterID.ValueString() == "" {
		return errors.ErrMissingClusterId
	}
	if plan.Name.IsNull() || plan.Name.ValueString() == "" {
		return errors.ErrMissingName
	}
	if plan.Type.IsNull() || plan.Type.ValueString() == "" {
		return errors.ErrMissingType
	}
	if plan.Configuration.IsNull() || plan.Configuration.ValueString() == "" {
		return errors.ErrMissingConfiguration
	}
	return nil
}

func validateWorkflowIDs(state providerschema.Workflow) error {
	if state.OrganizationID.IsNull() || state.OrganizationID.ValueString() == "" {
		return errors.ErrMissingOrganizationId
	}
	if state.ProjectID.IsNull() || state.ProjectID.ValueString() == "" {
		return errors.ErrMissingProjectId
	}
	if state.ClusterID.IsNull() || state.ClusterID.ValueString() == "" {
		return errors.ErrMissingClusterId
	}
	if state.ID.IsNull() || state.ID.ValueString() == "" {
		return errors.ErrMissingId
	}
	return nil
}

func workflowCollectionURL(host, organizationID, projectID, clusterID string) string {
	return fmt.Sprintf(
		"%s/v4/organizations/%s/projects/%s/clusters/%s/aiServices/workflows",
		host, organizationID, projectID, clusterID,
	)
}

func workflowURL(host, organizationID, projectID, clusterID, workflowID string) string {
	return fmt.Sprintf("%s/%s", workflowCollectionURL(host, organizationID, projectID, clusterID), workflowID)
}
