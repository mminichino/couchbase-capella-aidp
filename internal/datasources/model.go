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
	providerschema "github.com/mminichino/couchbase-capella-aidp/internal/schema"
)

var (
	_ datasource.DataSource              = &Model{}
	_ datasource.DataSourceWithConfigure = &Model{}
)

type modelDataSourceModel struct {
	OrganizationID   types.String         `tfsdk:"organization_id"`
	ID               types.String         `tfsdk:"id"`
	Name             types.String         `tfsdk:"name"`
	Status           types.String         `tfsdk:"status"`
	ConnectionString types.String         `tfsdk:"connection_string"`
	CloudConfig      types.Object         `tfsdk:"cloud_config"`
	Config           jsontypes.Normalized `tfsdk:"config"`
	Audit            types.Object         `tfsdk:"audit"`
}

// Model is the single-model data source.
type Model struct {
	*providerschema.Data
}

// NewModel is a helper function to simplify provider registration.
func NewModel() datasource.DataSource {
	return &Model{}
}

func (d *Model) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_model"
}

func (d *Model) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a Capella AI model by organization_id and id.",
		Attributes: map[string]schema.Attribute{
			"organization_id": schema.StringAttribute{Required: true},
			"id":              schema.StringAttribute{Required: true},
			"name":            schema.StringAttribute{Computed: true},
			"status":          schema.StringAttribute{Computed: true},
			"connection_string": schema.StringAttribute{
				Computed: true,
			},
			"cloud_config": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"provider": schema.StringAttribute{Computed: true},
					"region":   schema.StringAttribute{Computed: true},
					"compute": schema.SingleNestedAttribute{
						Computed: true,
						Attributes: map[string]schema.Attribute{
							"cpu":        schema.Int64Attribute{Computed: true},
							"gpu_memory": schema.Int64Attribute{Computed: true},
						},
					},
				},
			},
			"config": schema.StringAttribute{
				Computed:   true,
				CustomType: jsontypes.NormalizedType{},
			},
			"audit": computedAuditAttribute(),
		},
	}
}

func (d *Model) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*providerschema.Data)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *schema.Data, got: %T.", req.ProviderData),
		)
		return
	}
	d.Data = data
}

func (d *Model) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config modelDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := config.OrganizationID.ValueString()
	modelID := config.ID.ValueString()
	url := fmt.Sprintf("%s/v4/organizations/%s/aiServices/models/%s", d.HostURL, orgID, modelID)
	cfg := api.EndpointCfg{Url: url, Method: http.MethodGet, SuccessStatus: http.StatusOK}
	response, err := d.Client.ExecuteWithRetry(ctx, cfg, nil, d.Token, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error reading model", api.ParseError(err))
		return
	}

	var getResp api.GetModelResponse
	if err := json.Unmarshal(response.Body, &getResp); err != nil {
		resp.Diagnostics.AddError("Error reading model", "error unmarshalling response: "+err.Error())
		return
	}
	if getResp.Model == nil {
		resp.Diagnostics.AddError("Error reading model", "model response missing model payload")
		return
	}

	state := modelDetailsToDataSource(ctx, orgID, getResp.Model)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func modelDetailsToDataSource(ctx context.Context, orgID string, details *api.ModelDetails) modelDataSourceModel {
	out := modelDataSourceModel{
		ID:             types.StringValue(details.Id),
		OrganizationID: types.StringValue(orgID),
		Name:           types.StringValue(details.Name),
		Status:         types.StringValue(details.Status),
		CloudConfig:    providerschema.NewCloudConfigObject(ctx, details.CloudConfig),
		Audit:          providerschema.NewAuditObject(ctx, details.Audit),
	}
	if details.ConnectionString != nil {
		out.ConnectionString = types.StringValue(*details.ConnectionString)
	} else {
		out.ConnectionString = types.StringNull()
	}
	if len(details.Config) > 0 {
		out.Config = jsontypes.NewNormalizedValue(string(details.Config))
	} else {
		out.Config = jsontypes.NewNormalizedNull()
	}
	return out
}

func modelDetailsToSchema(ctx context.Context, orgID string, details *api.ModelDetails) providerschema.Model {
	out := providerschema.Model{
		ID:               types.StringValue(details.Id),
		OrganizationID:   types.StringValue(orgID),
		Name:             types.StringValue(details.Name),
		CatalogModelName: types.StringNull(),
		CloudConfig:      providerschema.NewCloudConfigObject(ctx, details.CloudConfig),
		Quantization:     types.StringNull(),
		Optimization:     types.StringNull(),
		Dimensions:       types.Int64Null(),
		Guardrails:       types.ListNull(types.StringType),
		Jailbreak:        jsontypes.NewNormalizedNull(),
		Caching:          jsontypes.NewNormalizedNull(),
		EnableBatching:   types.BoolNull(),
		KeywordFiltering: types.ListNull(types.StringType),
		Status:           types.StringValue(details.Status),
		Audit:            providerschema.NewAuditObject(ctx, details.Audit),
	}
	if details.ConnectionString != nil {
		out.ConnectionString = types.StringValue(*details.ConnectionString)
	} else {
		out.ConnectionString = types.StringNull()
	}
	if len(details.Config) > 0 {
		out.Config = jsontypes.NewNormalizedValue(string(details.Config))
	} else {
		out.Config = jsontypes.NewNormalizedNull()
	}
	return out
}
