package datasources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mminichino/couchbase-capella-aidp/internal/api"
	"github.com/mminichino/couchbase-capella-aidp/internal/errors"
	providerschema "github.com/mminichino/couchbase-capella-aidp/internal/schema"
)

var (
	_ datasource.DataSource              = &Workflow{}
	_ datasource.DataSourceWithConfigure = &Workflow{}
)

// Workflow is the single workflow data source implementation.
type Workflow struct {
	*providerschema.Data
}

// NewWorkflow is a helper function to simplify the provider implementation.
func NewWorkflow() datasource.DataSource {
	return &Workflow{}
}

func (d *Workflow) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflow"
}

func (d *Workflow) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a single AI Data Plane workflow by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required: true,
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
			"name": schema.StringAttribute{
				Computed: true,
			},
			"type": schema.StringAttribute{
				Computed: true,
			},
			"configuration": schema.StringAttribute{
				Computed:   true,
				CustomType: jsontypes.NormalizedType{},
			},
			"audit": computedAuditAttribute(),
		},
	}
}

func (d *Workflow) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *Workflow) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state providerschema.Workflow
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.OrganizationID.IsNull() || state.OrganizationID.ValueString() == "" {
		resp.Diagnostics.AddError("Error reading Capella workflow", errors.ErrMissingOrganizationId.Error())
		return
	}
	if state.ProjectID.IsNull() || state.ProjectID.ValueString() == "" {
		resp.Diagnostics.AddError("Error reading Capella workflow", errors.ErrMissingProjectId.Error())
		return
	}
	if state.ClusterID.IsNull() || state.ClusterID.ValueString() == "" {
		resp.Diagnostics.AddError("Error reading Capella workflow", errors.ErrMissingClusterId.Error())
		return
	}
	if state.ID.IsNull() || state.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Error reading Capella workflow", errors.ErrMissingId.Error())
		return
	}

	organizationID := state.OrganizationID.ValueString()
	projectID := state.ProjectID.ValueString()
	clusterID := state.ClusterID.ValueString()
	workflowID := state.ID.ValueString()

	url := fmt.Sprintf(
		"%s/v4/organizations/%s/projects/%s/clusters/%s/aiServices/workflows/%s",
		d.HostURL, organizationID, projectID, clusterID, workflowID,
	)
	cfg := api.EndpointCfg{Url: url, Method: http.MethodGet, SuccessStatus: http.StatusOK}
	response, err := d.Client.ExecuteWithRetry(ctx, cfg, nil, d.Token, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Capella workflow",
			"Could not read Capella workflow "+workflowID+": "+api.ParseError(err),
		)
		return
	}

	var getResp api.GetWorkflowResponse
	if err := json.Unmarshal(response.Body, &getResp); err != nil {
		resp.Diagnostics.AddError(
			"Error reading Capella workflow",
			"Could not unmarshal Capella workflow response: "+err.Error(),
		)
		return
	}

	id := getResp.Id
	if id == "" {
		id = workflowID
	}

	configuration := jsontypes.NewNormalizedNull()
	if len(getResp.Configuration) > 0 {
		configuration = jsontypes.NewNormalizedValue(string(getResp.Configuration))
	}

	state = providerschema.Workflow{
		ID:             types.StringValue(id),
		OrganizationID: types.StringValue(organizationID),
		ProjectID:      types.StringValue(projectID),
		ClusterID:      types.StringValue(clusterID),
		Name:           types.StringValue(getResp.Name),
		Type:           types.StringValue(getResp.Type),
		Configuration:  configuration,
		Audit:          providerschema.NewAuditObject(ctx, getResp.Audit),
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
