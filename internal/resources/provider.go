package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mminichino/couchbase-capella-aidp/internal/api"
	providerschema "github.com/mminichino/couchbase-capella-aidp/internal/schema"
)

var (
	_ resource.Resource                = &Provider{}
	_ resource.ResourceWithConfigure   = &Provider{}
	_ resource.ResourceWithImportState = &Provider{}
)

var allowedProviderTypes = map[string]struct{}{
	"awsS3":      {},
	"openAI":     {},
	"awsBedrock": {},
}

// Provider is the AI Data Plane provider resource implementation.
type Provider struct {
	*providerschema.Data
}

// NewProvider is a helper function to simplify provider implementation.
func NewProvider() resource.Resource {
	return &Provider{}
}

func (r *Provider) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_provider"
}

func (r *Provider) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = ProviderSchema()
}

func (r *Provider) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	data, ok := req.ProviderData.(*providerschema.Data)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *providerschema.Data, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.Data = data
}

func (r *Provider) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan providerschema.AIDPProvider
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateProviderPlan(plan); err != nil {
		resp.Diagnostics.AddError("Error creating Capella AIDP provider", err.Error())
		return
	}

	var configuration json.RawMessage
	if err := json.Unmarshal([]byte(plan.Configuration.ValueString()), &configuration); err != nil {
		resp.Diagnostics.AddError(
			"Error creating Capella AIDP provider",
			"Could not parse configuration JSON: "+err.Error(),
		)
		return
	}

	createReq := api.CreateProviderRequest{
		Type:          plan.Type.ValueString(),
		Name:          plan.Name.ValueString(),
		Configuration: configuration,
	}

	url := fmt.Sprintf(
		"%s/v4/organizations/%s/aiServices/providers",
		r.HostURL,
		plan.OrganizationID.ValueString(),
	)
	cfg := api.EndpointCfg{Url: url, Method: http.MethodPost, SuccessStatus: http.StatusCreated}
	response, err := r.Client.ExecuteWithRetry(ctx, cfg, createReq, r.Token, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating Capella AIDP provider",
			"Could not create provider: "+api.ParseError(err),
		)
		return
	}

	var createResp api.CreateProviderResponse
	if err := json.Unmarshal(response.Body, &createResp); err != nil {
		resp.Diagnostics.AddError(
			"Error creating Capella AIDP provider",
			"Could not parse create response: "+err.Error(),
		)
		return
	}

	refreshed, err := r.refreshProvider(ctx, plan.OrganizationID.ValueString(), createResp.Id, &plan)
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Provider created but failed to refresh state",
			"Provider creation succeeded, but reading the provider failed. "+
				"Run terraform plan to refresh. Unexpected error: "+api.ParseError(err),
		)
		plan.ID = types.StringValue(createResp.Id)
		plan.Audit = types.ObjectNull(providerschema.AuditAttrTypes())
		diags = resp.State.Set(ctx, &plan)
		resp.Diagnostics.Append(diags...)
		return
	}

	diags = resp.State.Set(ctx, refreshed)
	resp.Diagnostics.Append(diags...)
}

func (r *Provider) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state providerschema.AIDPProvider
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	refreshed, err := r.refreshProvider(ctx, state.OrganizationID.ValueString(), state.ID.ValueString(), &state)
	if err != nil {
		resourceNotFound, errString := api.CheckResourceNotFoundError(err)
		if resourceNotFound {
			tflog.Info(ctx, "AIDP provider does not exist remotely; removing from state")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading Capella AIDP provider",
			"Could not read provider "+state.ID.ValueString()+": "+errString,
		)
		return
	}

	diags = resp.State.Set(ctx, refreshed)
	resp.Diagnostics.Append(diags...)
}

func (r *Provider) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan providerschema.AIDPProvider
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state providerschema.AIDPProvider
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var configuration json.RawMessage
	if err := json.Unmarshal([]byte(plan.Configuration.ValueString()), &configuration); err != nil {
		resp.Diagnostics.AddError(
			"Error updating Capella AIDP provider",
			"Could not parse configuration JSON: "+err.Error(),
		)
		return
	}

	updateReq := api.UpdateProviderRequest{
		Configuration: configuration,
	}

	url := fmt.Sprintf(
		"%s/v4/organizations/%s/aiServices/providers/%s",
		r.HostURL,
		plan.OrganizationID.ValueString(),
		plan.ID.ValueString(),
	)

	headers := auditIfMatchHeader(ctx, state.Audit)
	cfg := api.EndpointCfg{Url: url, Method: http.MethodPut, SuccessStatus: http.StatusNoContent}
	_, err := executeAllowingStatuses(ctx, r.Client, cfg, updateReq, r.Token, headers, http.StatusOK)
	if err != nil {
		resourceNotFound, errString := api.CheckResourceNotFoundError(err)
		if resourceNotFound {
			tflog.Info(ctx, "AIDP provider does not exist remotely; removing from state")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error updating Capella AIDP provider",
			"Could not update provider "+plan.ID.ValueString()+": "+errString,
		)
		return
	}

	refreshed, err := r.refreshProvider(ctx, plan.OrganizationID.ValueString(), plan.ID.ValueString(), &plan)
	if err != nil {
		resourceNotFound, errString := api.CheckResourceNotFoundError(err)
		if resourceNotFound {
			tflog.Info(ctx, "AIDP provider does not exist remotely; removing from state")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error updating Capella AIDP provider",
			"Could not read provider after update: "+errString,
		)
		return
	}

	diags = resp.State.Set(ctx, refreshed)
	resp.Diagnostics.Append(diags...)
}

func (r *Provider) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state providerschema.AIDPProvider
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	url := fmt.Sprintf(
		"%s/v4/organizations/%s/aiServices/providers/%s",
		r.HostURL,
		state.OrganizationID.ValueString(),
		state.ID.ValueString(),
	)
	cfg := api.EndpointCfg{Url: url, Method: http.MethodDelete, SuccessStatus: http.StatusNoContent}
	_, err := executeAllowingStatuses(ctx, r.Client, cfg, nil, r.Token, nil, http.StatusOK)
	if err != nil {
		resourceNotFound, errString := api.CheckResourceNotFoundError(err)
		if resourceNotFound {
			tflog.Info(ctx, "AIDP provider already absent remotely; treating delete as success")
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting Capella AIDP provider",
			"Could not delete provider "+state.ID.ValueString()+": "+errString,
		)
		return
	}
}

func (r *Provider) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerschema.SplitImportID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import id format \"organizationId/providerId\": %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *Provider) getProvider(ctx context.Context, organizationID, providerID string) (*api.GetProviderResponse, error) {
	url := fmt.Sprintf(
		"%s/v4/organizations/%s/aiServices/providers/%s",
		r.HostURL,
		organizationID,
		providerID,
	)
	cfg := api.EndpointCfg{Url: url, Method: http.MethodGet, SuccessStatus: http.StatusOK}
	response, err := r.Client.ExecuteWithRetry(ctx, cfg, nil, r.Token, nil)
	if err != nil {
		return nil, err
	}

	var providerResp api.GetProviderResponse
	if err := json.Unmarshal(response.Body, &providerResp); err != nil {
		return nil, fmt.Errorf("error unmarshalling provider response: %w", err)
	}
	return &providerResp, nil
}

func (r *Provider) refreshProvider(
	ctx context.Context,
	organizationID, providerID string,
	fallback *providerschema.AIDPProvider,
) (*providerschema.AIDPProvider, error) {
	providerResp, err := r.getProvider(ctx, organizationID, providerID)
	if err != nil {
		return nil, err
	}

	state := &providerschema.AIDPProvider{
		ID:             types.StringValue(providerID),
		OrganizationID: types.StringValue(organizationID),
		Audit:          providerschema.NewAuditObject(ctx, providerResp.Audit),
	}

	if providerResp.Name != nil {
		state.Name = types.StringValue(*providerResp.Name)
	} else if fallback != nil {
		state.Name = fallback.Name
	} else {
		state.Name = types.StringNull()
	}

	if providerResp.Type != nil {
		state.Type = types.StringValue(*providerResp.Type)
	} else if fallback != nil {
		state.Type = fallback.Type
	} else {
		state.Type = types.StringNull()
	}

	if len(providerResp.Configuration) > 0 {
		state.Configuration = jsontypes.NewNormalizedValue(string(providerResp.Configuration))
	} else if fallback != nil {
		state.Configuration = fallback.Configuration
	} else {
		state.Configuration = jsontypes.NewNormalizedNull()
	}

	return state, nil
}

func validateProviderPlan(plan providerschema.AIDPProvider) error {
	if plan.OrganizationID.IsNull() || plan.OrganizationID.ValueString() == "" {
		return fmt.Errorf("organization_id is required")
	}
	if plan.Name.IsNull() || plan.Name.ValueString() == "" {
		return fmt.Errorf("name is required")
	}
	if plan.Type.IsNull() || plan.Type.ValueString() == "" {
		return fmt.Errorf("type is required")
	}
	if _, ok := allowedProviderTypes[plan.Type.ValueString()]; !ok {
		return fmt.Errorf("type must be one of awsS3, openAI, awsBedrock")
	}
	if plan.Configuration.IsNull() || plan.Configuration.IsUnknown() || plan.Configuration.ValueString() == "" {
		return fmt.Errorf("configuration is required")
	}
	return nil
}

func auditIfMatchHeader(ctx context.Context, auditObj types.Object) map[string]string {
	headers := make(map[string]string)
	if auditObj.IsNull() || auditObj.IsUnknown() {
		return headers
	}

	var audit providerschema.Audit
	diags := auditObj.As(ctx, &audit, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return headers
	}
	if !audit.Version.IsNull() && !audit.Version.IsUnknown() {
		headers["If-Match"] = strconv.FormatInt(audit.Version.ValueInt64(), 10)
	}
	return headers
}
