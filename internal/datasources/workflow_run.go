package datasources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mminichino/couchbase-capella-aidp/internal/api"
	"github.com/mminichino/couchbase-capella-aidp/internal/errors"
	providerschema "github.com/mminichino/couchbase-capella-aidp/internal/schema"
)

var (
	_ datasource.DataSource              = &WorkflowRun{}
	_ datasource.DataSourceWithConfigure = &WorkflowRun{}
)

// WorkflowRun is the single workflow run data source implementation.
type WorkflowRun struct {
	*providerschema.Data
}

// NewWorkflowRun is a helper function to simplify the provider implementation.
func NewWorkflowRun() datasource.DataSource {
	return &WorkflowRun{}
}

func (d *WorkflowRun) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflow_run"
}

func (d *WorkflowRun) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a single AI Data Plane workflow run by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Workflow run identifier.",
			},
			"organization_id": schema.StringAttribute{
				Required: true,
			},
			"project_id": schema.StringAttribute{
				Required: true,
			},
			"cluster_id": schema.StringAttribute{
				Required: true,
			},
			"workflow_id": schema.StringAttribute{
				Required: true,
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *WorkflowRun) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*providerschema.Data)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *providerschema.Data, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.Data = data
}

func (d *WorkflowRun) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state providerschema.WorkflowRun
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.OrganizationID.IsNull() || state.OrganizationID.ValueString() == "" {
		resp.Diagnostics.AddError("Error reading Capella workflow run", errors.ErrMissingOrganizationId.Error())
		return
	}
	if state.ProjectID.IsNull() || state.ProjectID.ValueString() == "" {
		resp.Diagnostics.AddError("Error reading Capella workflow run", errors.ErrMissingProjectId.Error())
		return
	}
	if state.ClusterID.IsNull() || state.ClusterID.ValueString() == "" {
		resp.Diagnostics.AddError("Error reading Capella workflow run", errors.ErrMissingClusterId.Error())
		return
	}
	if state.WorkflowID.IsNull() || state.WorkflowID.ValueString() == "" {
		resp.Diagnostics.AddError("Error reading Capella workflow run", errors.ErrMissingWorkflowId.Error())
		return
	}
	if state.ID.IsNull() || state.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Error reading Capella workflow run", errors.ErrMissingId.Error())
		return
	}

	organizationID := state.OrganizationID.ValueString()
	projectID := state.ProjectID.ValueString()
	clusterID := state.ClusterID.ValueString()
	workflowID := state.WorkflowID.ValueString()
	runID := state.ID.ValueString()

	url := fmt.Sprintf(
		"%s/v4/organizations/%s/projects/%s/clusters/%s/aiServices/workflows/%s/runs/%s",
		d.HostURL, organizationID, projectID, clusterID, workflowID, runID,
	)
	cfg := api.EndpointCfg{Url: url, Method: http.MethodGet, SuccessStatus: http.StatusOK}
	response, err := d.Client.ExecuteWithRetry(ctx, cfg, nil, d.Token, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Capella workflow run",
			"Could not read Capella workflow run "+runID+": "+api.ParseError(err),
		)
		return
	}

	var getResp api.GetWorkflowRunResponse
	if err := json.Unmarshal(response.Body, &getResp); err != nil {
		resp.Diagnostics.AddError(
			"Error reading Capella workflow run",
			"Could not unmarshal Capella workflow run response: "+err.Error(),
		)
		return
	}

	id := getResp.Id
	if id == "" {
		id = runID
	}

	status := types.StringNull()
	if getResp.Status != "" {
		status = types.StringValue(getResp.Status)
	}

	state = providerschema.WorkflowRun{
		ID:             types.StringValue(id),
		OrganizationID: types.StringValue(organizationID),
		ProjectID:      types.StringValue(projectID),
		ClusterID:      types.StringValue(clusterID),
		WorkflowID:     types.StringValue(workflowID),
		Status:         status,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
