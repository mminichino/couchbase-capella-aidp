package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mminichino/couchbase-capella-aidp/internal/api"
	"github.com/mminichino/couchbase-capella-aidp/internal/datasources"
	"github.com/mminichino/couchbase-capella-aidp/internal/resources"
	providerschema "github.com/mminichino/couchbase-capella-aidp/internal/schema"
	"github.com/mminichino/couchbase-capella-aidp/version"
)

var _ provider.Provider = &aidpProvider{}

const (
	capellaAuthenticationTokenField     = "authentication_token"
	capellaPublicAPIHostField           = "host"
	capellaGlobalAPIRequestTimeoutField = "global_api_request_timeout"
	apiRequestTimeout                   = 300 * time.Second
	defaultAPIHostURL                   = "https://cloudapi.cloud.couchbase.com"
	providerName                        = "couchbase-capella-aidp"
)

type aidpProvider struct {
	name string
}

// New is a helper function to simplify provider server and testing implementation.
func New() func() provider.Provider {
	return func() provider.Provider {
		return &aidpProvider{
			name: providerName,
		}
	}
}

func (p *aidpProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = p.name
	resp.Version = version.ProviderVersion
}

func (p *aidpProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for Couchbase Capella AI Data Plane (AIDP) APIs.",
		Attributes: map[string]schema.Attribute{
			capellaPublicAPIHostField: schema.StringAttribute{
				Optional:    true,
				Description: "Capella Public API HTTPS Host URL. May be set via the CAPELLA_HOST environment variable. Defaults to https://cloudapi.cloud.couchbase.com",
			},
			capellaAuthenticationTokenField: schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Capella API Token that serves as an authentication mechanism. May be set via the CAPELLA_AUTHENTICATION_TOKEN environment variable.",
			},
			capellaGlobalAPIRequestTimeoutField: schema.Int64Attribute{
				Optional:    true,
				Description: "Global API request timeout in seconds. May be set via the CAPELLA_GLOBAL_API_REQUEST_TIMEOUT environment variable. Defaults to 300. Value must be greater than or equal to 300.",
			},
		},
	}
}

func (p *aidpProvider) Configure(
	ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse,
) {
	tflog.Info(ctx, "Configuring the Capella AIDP Client")

	var config providerschema.Config
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.Host.IsNull() || (!config.Host.IsUnknown() && config.Host.ValueString() == "") {
		envHost, exists := os.LookupEnv("CAPELLA_HOST")
		if exists && envHost != "" {
			config.Host = types.StringValue(envHost)
		} else {
			config.Host = types.StringValue(defaultAPIHostURL)
		}
	}

	if config.AuthenticationToken.IsNull() || (!config.AuthenticationToken.IsUnknown() && config.AuthenticationToken.ValueString() == "") {
		if token := os.Getenv("CAPELLA_AUTHENTICATION_TOKEN"); token != "" {
			config.AuthenticationToken = types.StringValue(strings.Trim(strings.TrimSpace(token), `"'`))
		}
	}

	if config.AuthenticationToken.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root(capellaAuthenticationTokenField),
			"Unknown Capella Authentication Token",
			"The provider cannot create the Capella API client as there is an unknown configuration value for the Capella authentication token. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the CAPELLA_AUTHENTICATION_TOKEN environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	host := strings.TrimRight(strings.TrimSpace(config.Host.ValueString()), "/")
	authenticationToken := strings.TrimSpace(config.AuthenticationToken.ValueString())

	if host == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root(capellaPublicAPIHostField),
			"Missing Capella Public API Host",
			"The provider cannot create the Capella API client as there is a missing or empty value for the Capella API host. "+
				"Set the host value in the configuration or use the CAPELLA_HOST environment variable.",
		)
	}

	if authenticationToken == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root(capellaAuthenticationTokenField),
			"Missing Capella Authentication Token",
			"The provider cannot create the Capella API client as there is a missing or empty value for the Capella authentication token. "+
				"Set the authentication_token value in the configuration or use the CAPELLA_AUTHENTICATION_TOKEN environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, capellaPublicAPIHostField, host)
	ctx = tflog.SetField(ctx, capellaAuthenticationTokenField, authenticationToken)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, capellaAuthenticationTokenField)

	clientTimeout := apiRequestTimeout
	if !config.GlobalAPIRequestTimeout.IsNull() && !config.GlobalAPIRequestTimeout.IsUnknown() {
		clientTimeout = time.Duration(config.GlobalAPIRequestTimeout.ValueInt64()) * time.Second
	} else if t, found := os.LookupEnv("CAPELLA_GLOBAL_API_REQUEST_TIMEOUT"); found {
		seconds, err := strconv.Atoi(t)
		if err == nil {
			clientTimeout = time.Duration(seconds) * time.Second
		} else {
			tflog.Warn(ctx, fmt.Sprintf("Invalid client timeout value: %v", err))
		}
	}

	if clientTimeout < apiRequestTimeout {
		resp.Diagnostics.AddAttributeError(
			path.Root(capellaGlobalAPIRequestTimeoutField),
			"Invalid global API request timeout",
			fmt.Sprintf("global_api_request_timeout must be greater than or equal to %d seconds. Set via the provider config or CAPELLA_GLOBAL_API_REQUEST_TIMEOUT environment variable.", int64(apiRequestTimeout.Seconds())),
		)
		return
	}

	tflog.Debug(ctx, "Using HTTP client timeout", map[string]any{"seconds": int64(clientTimeout.Seconds())})

	client := api.NewClient(clientTimeout)
	providerData := &providerschema.Data{
		HostURL: host,
		Token:   authenticationToken,
		Client:  client,
	}

	resp.DataSourceData = providerData
	resp.ResourceData = providerData

	tflog.Info(ctx, "Configured Capella AIDP client", map[string]any{"success": true})
}

func (p *aidpProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewProvider,
		datasources.NewProviders,
		datasources.NewWorkflow,
		datasources.NewWorkflows,
		datasources.NewWorkflowRun,
		datasources.NewWorkflowRuns,
		datasources.NewModel,
		datasources.NewModels,
		datasources.NewModelAPIKey,
		datasources.NewModelAPIKeys,
		datasources.NewModelConnectionString,
	}
}

func (p *aidpProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewProvider,
		resources.NewWorkflow,
		resources.NewWorkflowRun,
		resources.NewModel,
		resources.NewModelActivation,
		resources.NewModelAPIKey,
	}
}
