package datasources

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mminichino/couchbase-capella-aidp/internal/api"
	"github.com/mminichino/couchbase-capella-aidp/internal/errors"
	providerschema "github.com/mminichino/couchbase-capella-aidp/internal/schema"
)

var (
	_ datasource.DataSource              = &WorkflowRuns{}
	_ datasource.DataSourceWithConfigure = &WorkflowRuns{}
)

type workflowRunsModel struct {
	OrganizationID types.String                 `tfsdk:"organization_id"`
	ProjectID      types.String                 `tfsdk:"project_id"`
	ClusterID      types.String                 `tfsdk:"cluster_id"`
	WorkflowID     types.String                 `tfsdk:"workflow_id"`
	Data           []providerschema.WorkflowRun `tfsdk:"data"`
}

// WorkflowRuns is the workflow runs list data source implementation.
type WorkflowRuns struct {
	*providerschema.Data
}

// NewWorkflowRuns is a helper function to simplify the provider implementation.
func NewWorkflowRuns() datasource.DataSource {
	return &WorkflowRuns{}
}

func (d *WorkflowRuns) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflow_runs"
}

func (d *WorkflowRuns) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists AI Data Plane workflow runs for a workflow.",
		Attributes: map[string]schema.Attribute{
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
			"data": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed: true,
						},
						"organization_id": schema.StringAttribute{
							Computed: true,
						},
						"project_id": schema.StringAttribute{
							Computed: true,
						},
						"cluster_id": schema.StringAttribute{
							Computed: true,
						},
						"workflow_id": schema.StringAttribute{
							Computed: true,
						},
						"status": schema.StringAttribute{
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func (d *WorkflowRuns) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *WorkflowRuns) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state workflowRunsModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.OrganizationID.IsNull() || state.OrganizationID.ValueString() == "" {
		resp.Diagnostics.AddError("Error reading Capella workflow runs", errors.ErrMissingOrganizationId.Error())
		return
	}
	if state.ProjectID.IsNull() || state.ProjectID.ValueString() == "" {
		resp.Diagnostics.AddError("Error reading Capella workflow runs", errors.ErrMissingProjectId.Error())
		return
	}
	if state.ClusterID.IsNull() || state.ClusterID.ValueString() == "" {
		resp.Diagnostics.AddError("Error reading Capella workflow runs", errors.ErrMissingClusterId.Error())
		return
	}
	if state.WorkflowID.IsNull() || state.WorkflowID.ValueString() == "" {
		resp.Diagnostics.AddError("Error reading Capella workflow runs", errors.ErrMissingWorkflowId.Error())
		return
	}

	organizationID := state.OrganizationID.ValueString()
	projectID := state.ProjectID.ValueString()
	clusterID := state.ClusterID.ValueString()
	workflowID := state.WorkflowID.ValueString()

	url := fmt.Sprintf(
		"%s/v4/organizations/%s/projects/%s/clusters/%s/aiServices/workflows/%s/runs",
		strings.TrimRight(d.HostURL, "/"), organizationID, projectID, clusterID, workflowID,
	)
	cfg := api.EndpointCfg{Url: url, Method: http.MethodGet, SuccessStatus: http.StatusOK}
	items, err := api.GetPaginated[[]api.GetWorkflowRunResponse](ctx, d.Client, d.Token, cfg)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Capella workflow runs",
			"Could not read Capella workflow runs: "+api.ParseError(err),
		)
		return
	}

	state.Data = make([]providerschema.WorkflowRun, 0, len(items))
	for _, item := range items {
		status := types.StringNull()
		if item.Status != "" {
			status = types.StringValue(item.Status)
		}
		state.Data = append(state.Data, providerschema.WorkflowRun{
			ID:             types.StringValue(item.Id),
			OrganizationID: types.StringValue(organizationID),
			ProjectID:      types.StringValue(projectID),
			ClusterID:      types.StringValue(clusterID),
			WorkflowID:     types.StringValue(workflowID),
			Status:         status,
		})
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
