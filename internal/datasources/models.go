package datasources

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mminichino/couchbase-capella-aidp/internal/api"
	providerschema "github.com/mminichino/couchbase-capella-aidp/internal/schema"
)

var (
	_ datasource.DataSource              = &Models{}
	_ datasource.DataSourceWithConfigure = &Models{}
)

type modelsDataSourceModel struct {
	OrganizationID types.String           `tfsdk:"organization_id"`
	ModelStatus    types.String           `tfsdk:"model_status"`
	ModelKind      types.String           `tfsdk:"model_kind"`
	Data           []providerschema.Model `tfsdk:"data"`
}

// Models lists Capella AI models in an organization.
type Models struct {
	*providerschema.Data
}

// NewModels is a helper function to simplify provider registration.
func NewModels() datasource.DataSource {
	return &Models{}
}

func (d *Models) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_models"
}

func (d *Models) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Capella AI models in an organization. " +
			"Optional `model_status` and `model_kind` query filters are supported.",
		Attributes: map[string]schema.Attribute{
			"organization_id": schema.StringAttribute{Required: true},
			"model_status": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Filter by model status (e.g. healthy, deploying, paused).",
			},
			"model_kind": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Filter by model kind: `text-generation` or `embedding-generation`.",
			},
			"data": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":                 schema.StringAttribute{Computed: true},
						"organization_id":    schema.StringAttribute{Computed: true},
						"name":               schema.StringAttribute{Computed: true},
						"catalog_model_name": schema.StringAttribute{Computed: true},
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
						"quantization":      schema.StringAttribute{Computed: true},
						"optimization":      schema.StringAttribute{Computed: true},
						"dimensions":        schema.Int64Attribute{Computed: true},
						"guardrails":        schema.ListAttribute{Computed: true, ElementType: types.StringType},
						"jailbreak":         schema.StringAttribute{Computed: true, CustomType: jsontypes.NormalizedType{}},
						"caching":           schema.StringAttribute{Computed: true, CustomType: jsontypes.NormalizedType{}},
						"enable_batching":   schema.BoolAttribute{Computed: true},
						"keyword_filtering": schema.ListAttribute{Computed: true, ElementType: types.StringType},
						"status":            schema.StringAttribute{Computed: true},
						"connection_string": schema.StringAttribute{Computed: true},
						"config":            schema.StringAttribute{Computed: true, CustomType: jsontypes.NormalizedType{}},
						"audit":             computedAuditAttribute(),
					},
				},
			},
		},
	}
}

func (d *Models) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *Models) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state modelsDataSourceModel
	diags := req.Config.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := strings.TrimSpace(state.OrganizationID.ValueString())
	endpoint := fmt.Sprintf("%s/v4/organizations/%s/aiServices/models", strings.TrimRight(d.HostURL, "/"), orgID)
	u, err := url.Parse(endpoint)
	if err != nil {
		resp.Diagnostics.AddError("Error listing models", err.Error())
		return
	}
	q := u.Query()
	if !state.ModelStatus.IsNull() && !state.ModelStatus.IsUnknown() && state.ModelStatus.ValueString() != "" {
		q.Set("modelStatus", state.ModelStatus.ValueString())
	}
	if !state.ModelKind.IsNull() && !state.ModelKind.IsUnknown() && state.ModelKind.ValueString() != "" {
		q.Set("modelKind", state.ModelKind.ValueString())
	}
	u.RawQuery = q.Encode()

	cfg := api.EndpointCfg{Url: u.String(), Method: http.MethodGet, SuccessStatus: http.StatusOK}
	items, err := api.GetPaginated[[]api.GetModelResponse](ctx, d.Client, d.Token, cfg)
	if err != nil {
		resp.Diagnostics.AddError("Error listing models", api.ParseError(err))
		return
	}

	state.Data = make([]providerschema.Model, 0, len(items))
	for _, item := range items {
		if item.Model == nil {
			continue
		}
		state.Data = append(state.Data, modelDetailsToSchema(ctx, orgID, item.Model))
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
