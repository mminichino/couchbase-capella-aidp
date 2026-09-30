package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mminichino/couchbase-capella-aidp/internal/api"
	providerschema "github.com/mminichino/couchbase-capella-aidp/internal/schema"
)

var (
	_ resource.Resource                = &ModelAPIKey{}
	_ resource.ResourceWithConfigure   = &ModelAPIKey{}
	_ resource.ResourceWithImportState = &ModelAPIKey{}
)

// ModelAPIKey is the Model Services API key resource.
type ModelAPIKey struct {
	*providerschema.Data
}

// NewModelAPIKey is a helper function to simplify provider registration.
func NewModelAPIKey() resource.Resource {
	return &ModelAPIKey{}
}

func (r *ModelAPIKey) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_model_api_key"
}

func (r *ModelAPIKey) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = ModelAPIKeySchema()
}

func (r *ModelAPIKey) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*providerschema.Data)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *schema.Data, got: %T.", req.ProviderData),
		)
		return
	}
	r.Data = data
}

func (r *ModelAPIKey) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan providerschema.ModelAPIKey
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq, err := buildCreateModelAPIKeyRequest(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Error creating model API key", err.Error())
		return
	}

	orgID := plan.OrganizationID.ValueString()
	url := fmt.Sprintf("%s/v4/organizations/%s/aiServices/models/apiKeys", strings.TrimRight(r.HostURL, "/"), orgID)
	cfg := api.EndpointCfg{
		Url:             url,
		Method:          http.MethodPost,
		SuccessStatus:   http.StatusCreated,
		SuccessStatuses: []int{http.StatusOK},
	}
	response, err := r.Client.ExecuteWithRetry(ctx, cfg, createReq, r.Token, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error creating model API key", api.ParseError(err))
		return
	}

	var createResp api.CreateModelAPIKeyResponse
	if err := json.Unmarshal(response.Body, &createResp); err != nil {
		resp.Diagnostics.AddError("Error creating model API key", "error unmarshalling response: "+err.Error())
		return
	}
	if createResp.Id == "" {
		resp.Diagnostics.AddError("Error creating model API key", "create response did not include an id")
		return
	}

	plan.ID = types.StringValue(createResp.Id)
	plan.Token = types.StringValue(createResp.Token)
	plan.Audit = nullAuditObject()
	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	refreshed, err := r.retrieveModelAPIKey(ctx, orgID, createResp.Id, plan)
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Model API key created but refresh incomplete",
			"Key was created but GET failed: "+api.ParseError(err)+
				". Token has been saved; run terraform apply -refresh-only to sync remaining attributes.",
		)
		return
	}
	refreshed.Token = types.StringValue(createResp.Token)
	// GET returns remaining lifetime; keep the configured expiry from the plan.
	refreshed.Expiry = plan.Expiry

	diags = resp.State.Set(ctx, refreshed)
	resp.Diagnostics.Append(diags...)
}

func (r *ModelAPIKey) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state providerschema.ModelAPIKey
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	keyID := state.ID.ValueString()
	if orgID == "" || keyID == "" {
		resp.Diagnostics.AddError("Error reading model API key", "organization_id and id are required")
		return
	}

	refreshed, err := r.retrieveModelAPIKey(ctx, orgID, keyID, state)
	if err != nil {
		notFound, errString := api.CheckResourceNotFoundError(err)
		if notFound {
			tflog.Info(ctx, "model API key not found remotely; removing from state")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading model API key", errString)
		return
	}

	// Token is only returned at create time.
	refreshed.Token = state.Token

	diags = resp.State.Set(ctx, refreshed)
	resp.Diagnostics.Append(diags...)
}

func (r *ModelAPIKey) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Unexpected Update",
		"Model API keys do not support in-place updates; all arguments force replacement.",
	)
}

func (r *ModelAPIKey) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state providerschema.ModelAPIKey
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	keyID := state.ID.ValueString()
	url := fmt.Sprintf("%s/v4/organizations/%s/aiServices/models/apiKeys/%s", r.HostURL, orgID, keyID)
	cfg := api.EndpointCfg{Url: url, Method: http.MethodDelete, SuccessStatus: http.StatusNoContent}
	_, err := r.Client.ExecuteWithRetry(ctx, cfg, nil, r.Token, nil)
	if err != nil {
		notFound, errString := api.CheckResourceNotFoundError(err)
		if notFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error deleting model API key", errString)
		return
	}
}

func (r *ModelAPIKey) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerschema.SplitImportID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Expected organizationId/apiKeyId: "+err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *ModelAPIKey) retrieveModelAPIKey(
	ctx context.Context, orgID, keyID string, base providerschema.ModelAPIKey,
) (*providerschema.ModelAPIKey, error) {
	url := fmt.Sprintf("%s/v4/organizations/%s/aiServices/models/apiKeys/%s", r.HostURL, orgID, keyID)
	cfg := api.EndpointCfg{Url: url, Method: http.MethodGet, SuccessStatus: http.StatusOK}
	response, err := r.Client.ExecuteWithRetry(ctx, cfg, nil, r.Token, nil)
	if err != nil {
		return nil, err
	}

	var getResp api.GetModelAPIKeyResponse
	if err := json.Unmarshal(response.Body, &getResp); err != nil {
		return nil, fmt.Errorf("error unmarshalling model API key: %w", err)
	}

	return mapModelAPIKeyToState(ctx, orgID, base, &getResp), nil
}

func mapModelAPIKeyToState(
	ctx context.Context, orgID string, base providerschema.ModelAPIKey, resp *api.GetModelAPIKeyResponse,
) *providerschema.ModelAPIKey {
	out := base
	out.OrganizationID = types.StringValue(orgID)
	if resp.KeyId != "" {
		out.ID = types.StringValue(resp.KeyId)
	}
	if resp.Name != "" {
		out.Name = types.StringValue(resp.Name)
	}
	if resp.Description != nil {
		out.Description = types.StringValue(*resp.Description)
	}
	// Capella GET returns remaining lifetime (a drifting float), not the configured
	// expiry days. Prefer the planned/configured value when we already have one.
	if !base.Expiry.IsNull() && !base.Expiry.IsUnknown() {
		out.Expiry = base.Expiry
	} else if resp.Expiry != nil {
		out.Expiry = types.Float64Value(float64(*resp.Expiry))
	}
	if resp.AllowedCIDRs != nil {
		out.AllowedCIDRs = providerschema.StringsToSet(resp.AllowedCIDRs)
	}
	if resp.AllowedModels != nil {
		ids := make([]string, 0, len(resp.AllowedModels))
		for _, m := range resp.AllowedModels {
			if m.Id != "" {
				ids = append(ids, m.Id)
			}
		}
		out.AllowedModels = stringsToList(ctx, ids)
	}
	if resp.Region != "" {
		out.Region = types.StringValue(resp.Region)
	}
	out.Audit = providerschema.NewAuditObject(ctx, resp.Audit)
	if out.Token.IsUnknown() {
		out.Token = types.StringNull()
	}
	return &out
}

func buildCreateModelAPIKeyRequest(ctx context.Context, plan providerschema.ModelAPIKey) (*api.CreateModelAPIKeyRequest, error) {
	var cidrs []string
	if !plan.AllowedCIDRs.IsNull() && !plan.AllowedCIDRs.IsUnknown() {
		diags := plan.AllowedCIDRs.ElementsAs(ctx, &cidrs, false)
		if diags.HasError() {
			return nil, fmt.Errorf("unable to convert allowed_cidrs")
		}
	}

	req := &api.CreateModelAPIKeyRequest{
		Name:         plan.Name.ValueString(),
		Expiry:       float32(plan.Expiry.ValueFloat64()),
		AllowedCIDRs: cidrs,
		Region:       plan.Region.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		v := plan.Description.ValueString()
		req.Description = &v
	}
	if !plan.AllowedModels.IsNull() && !plan.AllowedModels.IsUnknown() {
		models, err := listToStrings(ctx, plan.AllowedModels)
		if err != nil {
			return nil, err
		}
		req.AllowedModels = models
	}
	return req, nil
}
