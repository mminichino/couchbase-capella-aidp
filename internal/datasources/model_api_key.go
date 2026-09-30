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
	_ datasource.DataSource              = &ModelAPIKey{}
	_ datasource.DataSourceWithConfigure = &ModelAPIKey{}
)

// ModelAPIKey fetches a single Model Services API key.
type ModelAPIKey struct {
	*providerschema.Data
}

// NewModelAPIKey is a helper function to simplify provider registration.
func NewModelAPIKey() datasource.DataSource {
	return &ModelAPIKey{}
}

func (d *ModelAPIKey) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_model_api_key"
}

func (d *ModelAPIKey) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a Capella Model Services API key by organization_id and id. Token is not available.",
		Attributes: map[string]schema.Attribute{
			"organization_id": schema.StringAttribute{Required: true},
			"id":              schema.StringAttribute{Required: true},
			"name":            schema.StringAttribute{Computed: true},
			"description":     schema.StringAttribute{Computed: true},
			"expiry":          schema.Float64Attribute{Computed: true},
			"allowed_cidrs": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"allowed_models": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"region": schema.StringAttribute{Computed: true},
			"audit":  computedAuditAttribute(),
		},
	}
}

func (d *ModelAPIKey) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type modelAPIKeyDataSourceModel struct {
	OrganizationID types.String  `tfsdk:"organization_id"`
	ID             types.String  `tfsdk:"id"`
	Name           types.String  `tfsdk:"name"`
	Description    types.String  `tfsdk:"description"`
	Expiry         types.Float64 `tfsdk:"expiry"`
	AllowedCIDRs   types.Set     `tfsdk:"allowed_cidrs"`
	AllowedModels  types.List    `tfsdk:"allowed_models"`
	Region         types.String  `tfsdk:"region"`
	Audit          types.Object  `tfsdk:"audit"`
}

func (d *ModelAPIKey) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config modelAPIKeyDataSourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := config.OrganizationID.ValueString()
	keyID := config.ID.ValueString()
	url := fmt.Sprintf("%s/v4/organizations/%s/aiServices/models/apiKeys/%s", d.HostURL, orgID, keyID)
	cfg := api.EndpointCfg{Url: url, Method: http.MethodGet, SuccessStatus: http.StatusOK}
	response, err := d.Client.ExecuteWithRetry(ctx, cfg, nil, d.Token, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error reading model API key", api.ParseError(err))
		return
	}

	var getResp api.GetModelAPIKeyResponse
	if err := json.Unmarshal(response.Body, &getResp); err != nil {
		resp.Diagnostics.AddError("Error reading model API key", "error unmarshalling response: "+err.Error())
		return
	}

	state := mapAPIKeyResponse(ctx, orgID, &getResp)
	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func mapAPIKeyResponse(ctx context.Context, orgID string, resp *api.GetModelAPIKeyResponse) modelAPIKeyDataSourceModel {
	out := modelAPIKeyDataSourceModel{
		OrganizationID: types.StringValue(orgID),
		ID:             types.StringValue(resp.KeyId),
		Name:           types.StringValue(resp.Name),
		Region:         types.StringValue(resp.Region),
		Audit:          providerschema.NewAuditObject(ctx, resp.Audit),
	}
	if resp.Description != nil {
		out.Description = types.StringValue(*resp.Description)
	} else {
		out.Description = types.StringNull()
	}
	if resp.Expiry != nil {
		out.Expiry = types.Float64Value(float64(*resp.Expiry))
	} else {
		out.Expiry = types.Float64Null()
	}
	out.AllowedCIDRs = providerschema.StringsToSet(resp.AllowedCIDRs)
	if resp.AllowedModels != nil {
		ids := make([]string, 0, len(resp.AllowedModels))
		for _, m := range resp.AllowedModels {
			if m.Id != "" {
				ids = append(ids, m.Id)
			}
		}
		l, _ := types.ListValueFrom(ctx, types.StringType, ids)
		out.AllowedModels = l
	} else {
		out.AllowedModels = types.ListNull(types.StringType)
	}
	return out
}
