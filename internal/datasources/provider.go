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
	_ datasource.DataSource              = &Provider{}
	_ datasource.DataSourceWithConfigure = &Provider{}
)

// Provider is the AI Data Plane provider data source implementation.
type Provider struct {
	*providerschema.Data
}

// NewProvider is a helper function to simplify provider implementation.
func NewProvider() datasource.DataSource {
	return &Provider{}
}

func (d *Provider) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_provider"
}

func (d *Provider) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves an AI Data Plane provider integration by organization and provider ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the AI Data Plane provider integration.",
			},
			"organization_id": schema.StringAttribute{
				Required:    true,
				Description: "The GUID4 ID of the organization.",
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "The name of the provider integration.",
			},
			"type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The type of provider. One of `awsS3`, `openAI`, `awsBedrock`.",
			},
			"configuration": schema.StringAttribute{
				Computed:    true,
				CustomType:  jsontypes.NormalizedType{},
				Description: "JSON object containing the provider-specific configuration.",
			},
			"audit": computedAuditAttribute(),
		},
	}
}

func (d *Provider) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *Provider) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config providerschema.AIDPProvider
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.OrganizationID.IsNull() || config.OrganizationID.ValueString() == "" {
		resp.Diagnostics.AddError("Error reading Capella AIDP provider", "organization_id is required")
		return
	}
	if config.ID.IsNull() || config.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Error reading Capella AIDP provider", "id is required")
		return
	}

	organizationID := config.OrganizationID.ValueString()
	providerID := config.ID.ValueString()

	url := fmt.Sprintf(
		"%s/v4/organizations/%s/aiServices/providers/%s",
		d.HostURL,
		organizationID,
		providerID,
	)
	cfg := api.EndpointCfg{Url: url, Method: http.MethodGet, SuccessStatus: http.StatusOK}
	response, err := d.Client.ExecuteWithRetry(ctx, cfg, nil, d.Token, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Capella AIDP provider",
			"Could not read provider "+providerID+": "+api.ParseError(err),
		)
		return
	}

	var providerResp api.GetProviderResponse
	if err := json.Unmarshal(response.Body, &providerResp); err != nil {
		resp.Diagnostics.AddError(
			"Error reading Capella AIDP provider",
			"Could not parse provider response: "+err.Error(),
		)
		return
	}

	state := providerschema.AIDPProvider{
		ID:             types.StringValue(providerID),
		OrganizationID: types.StringValue(organizationID),
		Audit:          providerschema.NewAuditObject(ctx, providerResp.Audit),
	}

	if providerResp.Name != nil {
		state.Name = types.StringValue(*providerResp.Name)
	} else {
		state.Name = types.StringNull()
	}

	if providerResp.Type != nil {
		state.Type = types.StringValue(*providerResp.Type)
	} else {
		state.Type = types.StringNull()
	}

	if len(providerResp.Configuration) > 0 {
		state.Configuration = jsontypes.NewNormalizedValue(string(providerResp.Configuration))
	} else {
		state.Configuration = jsontypes.NewNormalizedNull()
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
