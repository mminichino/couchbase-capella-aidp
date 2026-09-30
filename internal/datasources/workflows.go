package datasources

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mminichino/couchbase-capella-aidp/internal/api"
	"github.com/mminichino/couchbase-capella-aidp/internal/errors"
	providerschema "github.com/mminichino/couchbase-capella-aidp/internal/schema"
)

var (
	_ datasource.DataSource              = &Workflows{}
	_ datasource.DataSourceWithConfigure = &Workflows{}
)

type workflowsModel struct {
	OrganizationID types.String              `tfsdk:"organization_id"`
	ProjectID      types.String              `tfsdk:"project_id"`
	ClusterID      types.String              `tfsdk:"cluster_id"`
	Data           []providerschema.Workflow `tfsdk:"data"`
}

// Workflows is the workflows list data source implementation.
type Workflows struct {
	*providerschema.Data
}

// NewWorkflows is a helper function to simplify the provider implementation.
func NewWorkflows() datasource.DataSource {
	return &Workflows{}
}

func (d *Workflows) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflows"
}

func (d *Workflows) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists AI Data Plane workflows for a Capella cluster.",
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
				},
			},
		},
	}
}

func (d *Workflows) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *Workflows) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state workflowsModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.OrganizationID.IsNull() || state.OrganizationID.ValueString() == "" {
		resp.Diagnostics.AddError("Error reading Capella workflows", errors.ErrMissingOrganizationId.Error())
		return
	}
	if state.ProjectID.IsNull() || state.ProjectID.ValueString() == "" {
		resp.Diagnostics.AddError("Error reading Capella workflows", errors.ErrMissingProjectId.Error())
		return
	}
	if state.ClusterID.IsNull() || state.ClusterID.ValueString() == "" {
		resp.Diagnostics.AddError("Error reading Capella workflows", errors.ErrMissingClusterId.Error())
		return
	}

	organizationID := state.OrganizationID.ValueString()
	projectID := state.ProjectID.ValueString()
	clusterID := state.ClusterID.ValueString()

	url := fmt.Sprintf(
		"%s/v4/organizations/%s/projects/%s/clusters/%s/aiServices/workflows",
		strings.TrimRight(d.HostURL, "/"), organizationID, projectID, clusterID,
	)
	cfg := api.EndpointCfg{Url: url, Method: http.MethodGet, SuccessStatus: http.StatusOK}
	items, err := api.GetPaginated[[]api.GetWorkflowResponse](ctx, d.Client, d.Token, cfg)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Capella workflows",
			"Could not read Capella workflows: "+api.ParseError(err),
		)
		return
	}

	state.Data = make([]providerschema.Workflow, 0, len(items))
	for _, item := range items {
		configuration := jsontypes.NewNormalizedNull()
		if len(item.Configuration) > 0 {
			configuration = jsontypes.NewNormalizedValue(string(item.Configuration))
		}
		state.Data = append(state.Data, providerschema.Workflow{
			ID:             types.StringValue(item.Id),
			OrganizationID: types.StringValue(organizationID),
			ProjectID:      types.StringValue(projectID),
			ClusterID:      types.StringValue(clusterID),
			Name:           types.StringValue(item.Name),
			Type:           types.StringValue(item.Type),
			Configuration:  configuration,
			Audit:          providerschema.NewAuditObject(ctx, item.Audit),
		})
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
