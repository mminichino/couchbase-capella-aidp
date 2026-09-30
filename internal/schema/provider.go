package schema

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mminichino/couchbase-capella-aidp/internal/api"
)

// Config maps provider schema data to a Go type.
type Config struct {
	Host                    types.String `tfsdk:"host"`
	AuthenticationToken     types.String `tfsdk:"authentication_token"`
	GlobalAPIRequestTimeout types.Int64  `tfsdk:"global_api_request_timeout"`
}

// Data is provider-defined data passed to resources and data sources.
type Data struct {
	Client  *api.Client
	HostURL string
	Token   string
}
