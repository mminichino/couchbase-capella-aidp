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
	_ datasource.DataSource              = &Providers{}
	_ datasource.DataSourceWithConfigure = &Providers{}
)

// Providers is the AI Data Plane providers list data source implementation.
type Providers struct {
	*providerschema.Data
}

type providersModel struct {
	OrganizationID types.String            `tfsdk:"organization_id"`
	ProviderType   types.String            `tfsdk:"provider_type"`
	Data           []providerListItemModel `tfsdk:"data"`
}

type providerListItemModel struct {
	ID            types.String         `tfsdk:"id"`
	Name          types.String         `tfsdk:"name"`
	Type          types.String         `tfsdk:"type"`
	Configuration jsontypes.Normalized `tfsdk:"configuration"`
	Audit         types.Object         `tfsdk:"audit"`
}

// NewProviders is a helper function to simplify provider implementation.
func NewProviders() datasource.DataSource {
	return &Providers{}
}

func (d *Providers) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_providers"
}

func (d *Providers) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists AI Data Plane provider integrations for an organization. " +
			"Optionally filter by `provider_type` (`awsS3`, `openAI`, `awsBedrock`).",
		Attributes: map[string]schema.Attribute{
			"organization_id": schema.StringAttribute{
				Required:    true,
				Description: "The GUID4 ID of the organization.",
			},
			"provider_type": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional filter for provider type. Valid values: `awsS3`, `openAI`, `awsBedrock`.",
			},
			"data": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The list of AI Data Plane provider integrations.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "The ID of the provider integration.",
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
				},
			},
		},
	}
}

func (d *Providers) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *Providers) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config providersModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.OrganizationID.IsNull() || config.OrganizationID.ValueString() == "" {
		resp.Diagnostics.AddError("Error reading Capella AIDP providers", "organization_id is required")
		return
	}

	organizationID := config.OrganizationID.ValueString()
	endpoint := fmt.Sprintf(
		"%s/v4/organizations/%s/aiServices/providers",
		strings.TrimRight(d.HostURL, "/"),
		organizationID,
	)

	if !config.ProviderType.IsNull() && !config.ProviderType.IsUnknown() && config.ProviderType.ValueString() != "" {
		endpoint = endpoint + "?providerType=" + url.QueryEscape(config.ProviderType.ValueString())
	}

	cfg := api.EndpointCfg{Url: endpoint, Method: http.MethodGet, SuccessStatus: http.StatusOK}
	items, err := api.GetPaginated[[]api.ProviderListItem](ctx, d.Client, d.Token, cfg)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading Capella AIDP providers",
			"Could not list providers: "+api.ParseError(err),
		)
		return
	}

	state := providersModel{
		OrganizationID: config.OrganizationID,
		ProviderType:   config.ProviderType,
		Data:           make([]providerListItemModel, 0, len(items)),
	}

	for _, item := range items {
		entry := providerListItemModel{
			ID:    types.StringValue(item.Id),
			Name:  types.StringValue(item.Name),
			Type:  types.StringValue(item.Type),
			Audit: providerschema.NewAuditObject(ctx, item.Audit),
		}
		if len(item.Configuration) > 0 {
			entry.Configuration = jsontypes.NewNormalizedValue(string(item.Configuration))
		} else {
			entry.Configuration = jsontypes.NewNormalizedNull()
		}
		state.Data = append(state.Data, entry)
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}
