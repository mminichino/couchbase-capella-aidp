package datasources

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mminichino/couchbase-capella-aidp/internal/api"
	providerschema "github.com/mminichino/couchbase-capella-aidp/internal/schema"
)

var (
	_ datasource.DataSource              = &ModelAPIKeys{}
	_ datasource.DataSourceWithConfigure = &ModelAPIKeys{}
)

type modelAPIKeysDataSourceModel struct {
	OrganizationID types.String                 `tfsdk:"organization_id"`
	FilterBy       types.String                 `tfsdk:"filter_by"`
	Data           []modelAPIKeyDataSourceModel `tfsdk:"data"`
}

// ModelAPIKeys lists Model Services API keys.
type ModelAPIKeys struct {
	*providerschema.Data
}

// NewModelAPIKeys is a helper function to simplify provider registration.
func NewModelAPIKeys() datasource.DataSource {
	return &ModelAPIKeys{}
}

func (d *ModelAPIKeys) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_model_api_keys"
}

func (d *ModelAPIKeys) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Capella Model Services API keys. " +
			"Optional `filter_by` supports values such as `region:eq:us-east-1`.",
		Attributes: map[string]schema.Attribute{
			"organization_id": schema.StringAttribute{Required: true},
			"filter_by": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Filter criteria, e.g. `region:eq:us-east-1`.",
			},
			"data": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"organization_id": schema.StringAttribute{Computed: true},
						"id":              schema.StringAttribute{Computed: true},
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
				},
			},
		},
	}
}

func (d *ModelAPIKeys) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ModelAPIKeys) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state modelAPIKeysDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	endpoint := fmt.Sprintf("%s/v4/organizations/%s/aiServices/models/apiKeys", strings.TrimRight(d.HostURL, "/"), orgID)
	u, err := url.Parse(endpoint)
	if err != nil {
		resp.Diagnostics.AddError("Error listing model API keys", err.Error())
		return
	}
	q := u.Query()
	if !state.FilterBy.IsNull() && !state.FilterBy.IsUnknown() && state.FilterBy.ValueString() != "" {
		q.Set("filterBy", state.FilterBy.ValueString())
	}
	u.RawQuery = q.Encode()

	cfg := api.EndpointCfg{Url: u.String(), Method: http.MethodGet, SuccessStatus: http.StatusOK}
	items, err := api.GetPaginated[[]api.GetModelAPIKeyResponse](ctx, d.Client, d.Token, cfg)
	if err != nil {
		resp.Diagnostics.AddError("Error listing model API keys", api.ParseError(err))
		return
	}

	state.Data = make([]modelAPIKeyDataSourceModel, 0, len(items))
	for i := range items {
		state.Data = append(state.Data, mapAPIKeyResponse(ctx, orgID, &items[i]))
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
