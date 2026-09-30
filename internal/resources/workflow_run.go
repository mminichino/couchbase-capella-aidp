package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mminichino/couchbase-capella-aidp/internal/api"
	"github.com/mminichino/couchbase-capella-aidp/internal/errors"
	providerschema "github.com/mminichino/couchbase-capella-aidp/internal/schema"
)

var (
	_ resource.Resource                = &WorkflowRun{}
	_ resource.ResourceWithConfigure   = &WorkflowRun{}
	_ resource.ResourceWithImportState = &WorkflowRun{}
)

const errorMessageWhileWorkflowRunCreation = "There is an error during workflow run creation. Please check in Capella to see if any hanging resources" +
	" have been created, unexpected error: "

const errorMessageAfterWorkflowRunCreation = "Workflow run creation is successful, but encountered an error while checking the current" +
	" state of the workflow run. Please run `terraform plan` after 1-2 minutes to know the" +
	" current workflow run state. Additionally, run `terraform apply --refresh-only` to update" +
	" the state from remote, unexpected error: "

// WorkflowRun is the AI Data Plane workflow run resource implementation.
type WorkflowRun struct {
	*providerschema.Data
}

// NewWorkflowRun is a helper function to simplify the provider implementation.
func NewWorkflowRun() resource.Resource {
	return &WorkflowRun{}
}

func (r *WorkflowRun) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflow_run"
}

func (r *WorkflowRun) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = WorkflowRunSchema()
}

func (r *WorkflowRun) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WorkflowRun) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan providerschema.WorkflowRun
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateCreateWorkflowRun(plan); err != nil {
		resp.Diagnostics.AddError(
			"Error creating workflow run",
			"Could not create workflow run, unexpected error: "+err.Error(),
		)
		return
	}

	url := workflowRunsURL(
		r.HostURL,
		plan.OrganizationID.ValueString(),
		plan.ProjectID.ValueString(),
		plan.ClusterID.ValueString(),
		plan.WorkflowID.ValueString(),
	)
	cfg := api.EndpointCfg{Url: url, Method: http.MethodPost, SuccessStatus: http.StatusCreated}
	response, err := executeAllowingStatuses(ctx, r.Client, cfg, nil, r.Token, nil, http.StatusAccepted)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating workflow run",
			errorMessageWhileWorkflowRunCreation+api.ParseError(err),
		)
		return
	}

	var createResp api.CreateWorkflowRunResponse
	if err := json.Unmarshal(response.Body, &createResp); err != nil {
		resp.Diagnostics.AddError(
			"Error creating workflow run",
			errorMessageWhileWorkflowRunCreation+"error during unmarshalling: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, initializeWorkflowRunWithPlanAndID(plan, createResp.Id))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	refreshed, err := r.refreshWorkflowRun(
		ctx,
		plan.OrganizationID.ValueString(),
		plan.ProjectID.ValueString(),
		plan.ClusterID.ValueString(),
		plan.WorkflowID.ValueString(),
		createResp.Id,
	)
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Error reading Capella workflow run",
			errorMessageAfterWorkflowRunCreation+api.ParseError(err),
		)
		return
	}

	diags = resp.State.Set(ctx, refreshed)
	resp.Diagnostics.Append(diags...)
}

func (r *WorkflowRun) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state providerschema.WorkflowRun
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateWorkflowRunIDs(state); err != nil {
		resp.Diagnostics.AddError(
			"Error reading Capella workflow run",
			"Could not read Capella workflow run: "+err.Error(),
		)
		return
	}

	refreshed, err := r.refreshWorkflowRun(
		ctx,
		state.OrganizationID.ValueString(),
		state.ProjectID.ValueString(),
		state.ClusterID.ValueString(),
		state.WorkflowID.ValueString(),
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
			"Error reading Capella workflow run",
			"Could not read Capella workflow run "+state.ID.ValueString()+": "+errString,
		)
		return
	}

	diags = resp.State.Set(ctx, refreshed)
	resp.Diagnostics.Append(diags...)
}

func (r *WorkflowRun) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// Capella does not support updating workflow runs.
}

func (r *WorkflowRun) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state providerschema.WorkflowRun
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateWorkflowRunIDs(state); err != nil {
		resp.Diagnostics.AddError(
			"Error stopping Capella workflow run",
			"Could not stop Capella workflow run: "+err.Error(),
		)
		return
	}

	// Stop endpoint has no runId in the path.
	url := workflowRunsURL(
		r.HostURL,
		state.OrganizationID.ValueString(),
		state.ProjectID.ValueString(),
		state.ClusterID.ValueString(),
		state.WorkflowID.ValueString(),
	)
	cfg := api.EndpointCfg{Url: url, Method: http.MethodDelete, SuccessStatus: http.StatusAccepted}
	_, err := executeAllowingStatuses(ctx, r.Client, cfg, nil, r.Token, nil, http.StatusNoContent)
	if err != nil {
		if isNotFoundOrConflict(err) {
			tflog.Info(ctx, "workflow run already finished or not found; removing resource from state")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error stopping Capella workflow run",
			"Could not stop Capella workflow run "+state.ID.ValueString()+": "+api.ParseError(err),
		)
		return
	}
}

func (r *WorkflowRun) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerschema.SplitImportID(req.ID, 5)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: organizationId/projectId/clusterId/workflowId/runId. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("cluster_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workflow_id"), parts[3])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[4])...)
}

func (r *WorkflowRun) refreshWorkflowRun(
	ctx context.Context, organizationID, projectID, clusterID, workflowID, runID string,
) (*providerschema.WorkflowRun, error) {
	url := workflowRunURL(r.HostURL, organizationID, projectID, clusterID, workflowID, runID)
	cfg := api.EndpointCfg{Url: url, Method: http.MethodGet, SuccessStatus: http.StatusOK}
	response, err := r.Client.ExecuteWithRetry(ctx, cfg, nil, r.Token, nil)
	if err != nil {
		return nil, err
	}

	var getResp api.GetWorkflowRunResponse
	if err := json.Unmarshal(response.Body, &getResp); err != nil {
		return nil, fmt.Errorf("error unmarshalling workflow run response: %w", err)
	}

	id := getResp.Id
	if id == "" {
		id = runID
	}

	status := types.StringNull()
	if getResp.Status != "" {
		status = types.StringValue(getResp.Status)
	}

	return &providerschema.WorkflowRun{
		ID:             types.StringValue(id),
		OrganizationID: types.StringValue(organizationID),
		ProjectID:      types.StringValue(projectID),
		ClusterID:      types.StringValue(clusterID),
		WorkflowID:     types.StringValue(workflowID),
		Status:         status,
	}, nil
}

func initializeWorkflowRunWithPlanAndID(plan providerschema.WorkflowRun, id string) providerschema.WorkflowRun {
	plan.ID = types.StringValue(id)
	plan.Status = types.StringNull()
	return plan
}

func validateCreateWorkflowRun(plan providerschema.WorkflowRun) error {
	if plan.OrganizationID.IsNull() || plan.OrganizationID.ValueString() == "" {
		return errors.ErrMissingOrganizationId
	}
	if plan.ProjectID.IsNull() || plan.ProjectID.ValueString() == "" {
		return errors.ErrMissingProjectId
	}
	if plan.ClusterID.IsNull() || plan.ClusterID.ValueString() == "" {
		return errors.ErrMissingClusterId
	}
	if plan.WorkflowID.IsNull() || plan.WorkflowID.ValueString() == "" {
		return errors.ErrMissingWorkflowId
	}
	return nil
}

func validateWorkflowRunIDs(state providerschema.WorkflowRun) error {
	if err := validateCreateWorkflowRun(state); err != nil {
		return err
	}
	if state.ID.IsNull() || state.ID.ValueString() == "" {
		return errors.ErrMissingId
	}
	return nil
}

func workflowRunsURL(host, organizationID, projectID, clusterID, workflowID string) string {
	return fmt.Sprintf(
		"%s/v4/organizations/%s/projects/%s/clusters/%s/aiServices/workflows/%s/runs",
		host, organizationID, projectID, clusterID, workflowID,
	)
}

func workflowRunURL(host, organizationID, projectID, clusterID, workflowID, runID string) string {
	return fmt.Sprintf("%s/%s", workflowRunsURL(host, organizationID, projectID, clusterID, workflowID), runID)
}
