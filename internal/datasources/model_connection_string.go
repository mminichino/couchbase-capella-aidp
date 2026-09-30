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
	providerschema "github.com/mminichino/couchbase-capella-aidp/internal/schema"
)

var (
	_ datasource.DataSource              = &ModelConnectionString{}
	_ datasource.DataSourceWithConfigure = &ModelConnectionString{}
)

type modelConnectionStringModel struct {
	OrganizationID   types.String `tfsdk:"organization_id"`
	ModelID          types.String `tfsdk:"model_id"`
	ConnectionString types.String `tfsdk:"connection_string"`
}

// ModelConnectionString fetches a model's connection string.
type ModelConnectionString struct {
	*providerschema.Data
}

// NewModelConnectionString is a helper function to simplify provider registration.
func NewModelConnectionString() datasource.DataSource {
	return &ModelConnectionString{}
}

func (d *ModelConnectionString) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_model_connection_string"
}

func (d *ModelConnectionString) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches the connection string for a Capella AI model.",
		Attributes: map[string]schema.Attribute{
			"organization_id": schema.StringAttribute{Required: true},
			"model_id":        schema.StringAttribute{Required: true},
			"connection_string": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *ModelConnectionString) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ModelConnectionString) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config modelConnectionStringModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := config.OrganizationID.ValueString()
	modelID := config.ModelID.ValueString()
	url := fmt.Sprintf("%s/v4/organizations/%s/aiServices/models/%s/connectionString", d.HostURL, orgID, modelID)
	cfg := api.EndpointCfg{Url: url, Method: http.MethodGet, SuccessStatus: http.StatusOK}
	response, err := d.Client.ExecuteWithRetry(ctx, cfg, nil, d.Token, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error reading model connection string", api.ParseError(err))
		return
	}

	var getResp api.GetConnectionStringResponse
	if err := json.Unmarshal(response.Body, &getResp); err != nil {
		resp.Diagnostics.AddError("Error reading model connection string", "error unmarshalling response: "+err.Error())
		return
	}

	state := modelConnectionStringModel{
		OrganizationID: types.StringValue(orgID),
		ModelID:        types.StringValue(modelID),
	}
	if getResp.ConnectionString != nil {
		state.ConnectionString = types.StringValue(*getResp.ConnectionString)
	} else {
		state.ConnectionString = types.StringNull()
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}
