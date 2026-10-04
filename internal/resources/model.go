package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	_ resource.Resource                = &Model{}
	_ resource.ResourceWithConfigure   = &Model{}
	_ resource.ResourceWithImportState = &Model{}
)

const (
	modelDeployPollInterval = 30 * time.Second
	modelDeployMaxAttempts  = 60
)

// Model is the Capella AI model resource.
type Model struct {
	*providerschema.Data
}

// NewModel is a helper function to simplify provider registration.
func NewModel() resource.Resource {
	return &Model{}
}

func (r *Model) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_model"
}

func (r *Model) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = ModelSchema()
}

func (r *Model) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *Model) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan providerschema.Model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq, err := buildCreateModelRequest(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Error creating model", err.Error())
		return
	}

	orgID := plan.OrganizationID.ValueString()
	url := fmt.Sprintf("%s/v4/organizations/%s/aiServices/models", r.HostURL, orgID)
	cfg := api.EndpointCfg{Url: url, Method: http.MethodPost, SuccessStatus: http.StatusAccepted}
	response, err := r.Client.ExecuteWithRetry(ctx, cfg, createReq, r.Token, nil)
	if err != nil {
		resp.Diagnostics.AddError("Error creating model", api.ParseError(err))
		return
	}

	var createResp api.CreateModelResponse
	if err := json.Unmarshal(response.Body, &createResp); err != nil {
		resp.Diagnostics.AddError("Error creating model", "error unmarshalling response: "+err.Error())
		return
	}
	if createResp.Id == "" {
		resp.Diagnostics.AddError("Error creating model", "create response did not include an id")
		return
	}

	plan.ID = types.StringValue(createResp.Id)
	diags = resp.State.Set(ctx, initializeModelComputed(plan))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	refreshed, err := r.waitForModelDeploy(ctx, orgID, createResp.Id, plan)
	if err != nil {
		if refreshed != nil {
			// Keep failed/partial model in state so terraform destroy can clean it up.
			_ = resp.State.Set(ctx, refreshed)
		}
		resp.Diagnostics.AddError("Error creating model", err.Error())
		return
	}

	diags = resp.State.Set(ctx, refreshed)
	resp.Diagnostics.Append(diags...)
}

func (r *Model) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state providerschema.Model
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	modelID := state.ID.ValueString()
	if orgID == "" || modelID == "" {
		resp.Diagnostics.AddError("Error reading model", "organization_id and id are required")
		return
	}

	refreshed, err := r.retrieveModel(ctx, orgID, modelID, state)
	if err != nil {
		notFound, errString := api.CheckResourceNotFoundError(err)
		if notFound {
			tflog.Info(ctx, "model not found remotely; removing from state")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading model", errString)
		return
	}

	diags = resp.State.Set(ctx, refreshed)
	resp.Diagnostics.Append(diags...)
}

func (r *Model) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan providerschema.Model
	var state providerschema.Model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := plan.OrganizationID.ValueString()
	modelID := plan.ID.ValueString()

	updateReq, err := buildUpdateModelRequest(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Error updating model", err.Error())
		return
	}

	headers := ifMatchHeadersFromAudit(state.Audit)
	url := fmt.Sprintf("%s/v4/organizations/%s/aiServices/models/%s", r.HostURL, orgID, modelID)
	cfg := api.EndpointCfg{
		Url:             url,
		Method:          http.MethodPut,
		SuccessStatus:   http.StatusNoContent,
		SuccessStatuses: []int{http.StatusOK, http.StatusAccepted},
	}
	_, err = r.Client.ExecuteWithRetry(ctx, cfg, updateReq, r.Token, headers)
	if err != nil {
		notFound, errString := api.CheckResourceNotFoundError(err)
		if notFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error updating model", errString)
		return
	}

	refreshed, err := r.retrieveModel(ctx, orgID, modelID, plan)
	if err != nil {
		resp.Diagnostics.AddError("Error reading model after update", api.ParseError(err))
		return
	}

	diags = resp.State.Set(ctx, refreshed)
	resp.Diagnostics.Append(diags...)
}

func (r *Model) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state providerschema.Model
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	modelID := state.ID.ValueString()
	if err := r.deleteModel(ctx, orgID, modelID); err != nil {
		resp.Diagnostics.AddError("Error deleting model", err.Error())
		return
	}
	if err := r.waitForModelDeleted(ctx, orgID, modelID); err != nil {
		resp.Diagnostics.AddError("Error deleting model", err.Error())
		return
	}
}

func (r *Model) deleteModel(ctx context.Context, orgID, modelID string) error {
	url := fmt.Sprintf("%s/v4/organizations/%s/aiServices/models/%s", r.HostURL, orgID, modelID)
	cfg := api.EndpointCfg{
		Url:             url,
		Method:          http.MethodDelete,
		SuccessStatus:   http.StatusAccepted,
		SuccessStatuses: []int{http.StatusNoContent},
	}

	for attempt := 0; attempt < modelDeployMaxAttempts; attempt++ {
		_, err := r.Client.ExecuteWithRetry(ctx, cfg, nil, r.Token, nil)
		if err == nil {
			return nil
		}
		notFound, errString := api.CheckResourceNotFoundError(err)
		if notFound {
			return nil
		}
		if !isModelInUseError(err) {
			return fmt.Errorf("%s", errString)
		}
		tflog.Info(ctx, "model still in use; retrying delete", map[string]any{
			"model_id": modelID,
			"attempt":  attempt + 1,
		})
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(modelDeployPollInterval):
		}
	}
	return fmt.Errorf("timed out waiting to delete model %s still referenced by another model", modelID)
}

func (r *Model) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerschema.SplitImportID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Expected organizationId/modelId: "+err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

func (r *Model) waitForModelDeploy(
	ctx context.Context, orgID, modelID string, base providerschema.Model,
) (*providerschema.Model, error) {
	var last *providerschema.Model
	var lastErr error
	for attempt := 0; attempt < modelDeployMaxAttempts; attempt++ {
		refreshed, err := r.retrieveModel(ctx, orgID, modelID, base)
		if err != nil {
			lastErr = err
			tflog.Warn(ctx, "error polling model status", map[string]any{"error": api.ParseError(err)})
		} else {
			last = refreshed
			lastErr = nil
			status := strings.ToLower(refreshed.Status.ValueString())
			if status == "healthy" {
				return refreshed, nil
			}
			if status == "deployfailed" || status == "failed" {
				msg := fmt.Sprintf("model %s deployment failed with status %s", modelID, refreshed.Status.ValueString())
				if !refreshed.Config.IsNull() && !refreshed.Config.IsUnknown() && refreshed.Config.ValueString() != "" {
					msg += "; config=" + refreshed.Config.ValueString()
				}
				msg += " (Capella accepted the create request but async provisioning failed; " +
					"quantization/optimization are not supported on every catalog model)"
				return refreshed, fmt.Errorf("%s", msg)
			}
			tflog.Info(ctx, "waiting for model deployment", map[string]any{
				"status":  refreshed.Status.ValueString(),
				"attempt": attempt + 1,
			})
		}
		select {
		case <-ctx.Done():
			if last != nil {
				return last, ctx.Err()
			}
			return nil, ctx.Err()
		case <-time.After(modelDeployPollInterval):
		}
	}
	if last != nil {
		return last, fmt.Errorf("timed out waiting for model %s to become healthy (last status %q)", modelID, last.Status.ValueString())
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("timed out waiting for model %s to become healthy", modelID)
}

func (r *Model) waitForModelDeleted(ctx context.Context, orgID, modelID string) error {
	for attempt := 0; attempt < modelDeployMaxAttempts; attempt++ {
		url := fmt.Sprintf("%s/v4/organizations/%s/aiServices/models/%s", r.HostURL, orgID, modelID)
		cfg := api.EndpointCfg{Url: url, Method: http.MethodGet, SuccessStatus: http.StatusOK}
		_, err := r.Client.ExecuteWithRetry(ctx, cfg, nil, r.Token, nil)
		if err != nil {
			notFound, _ := api.CheckResourceNotFoundError(err)
			if notFound {
				return nil
			}
			tflog.Warn(ctx, "error polling model delete", map[string]any{"error": api.ParseError(err)})
		} else {
			tflog.Info(ctx, "waiting for model delete", map[string]any{
				"model_id": modelID,
				"attempt":  attempt + 1,
			})
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(modelDeployPollInterval):
		}
	}
	return fmt.Errorf("timed out waiting for model %s to be deleted", modelID)
}

func isModelInUseError(err error) bool {
	msg := strings.ToLower(api.ParseError(err))
	return strings.Contains(msg, "embedding model in use") || strings.Contains(msg, `"code":14050`)
}

func (r *Model) retrieveModel(
	ctx context.Context, orgID, modelID string, base providerschema.Model,
) (*providerschema.Model, error) {
	url := fmt.Sprintf("%s/v4/organizations/%s/aiServices/models/%s", r.HostURL, orgID, modelID)
	cfg := api.EndpointCfg{Url: url, Method: http.MethodGet, SuccessStatus: http.StatusOK}
	response, err := r.Client.ExecuteWithRetry(ctx, cfg, nil, r.Token, nil)
	if err != nil {
		return nil, err
	}

	var getResp api.GetModelResponse
	if err := json.Unmarshal(response.Body, &getResp); err != nil {
		return nil, fmt.Errorf("error unmarshalling model: %w", err)
	}
	if getResp.Model == nil {
		return nil, fmt.Errorf("model response missing model payload")
	}

	return mapModelToState(ctx, orgID, base, getResp.Model), nil
}

func mapModelToState(
	ctx context.Context, orgID string, base providerschema.Model, details *api.ModelDetails,
) *providerschema.Model {
	out := base
	out.OrganizationID = types.StringValue(orgID)
	if details.Id != "" {
		out.ID = types.StringValue(details.Id)
	}
	if details.Name != "" {
		out.Name = types.StringValue(details.Name)
	}
	out.Status = types.StringValue(details.Status)
	if details.ConnectionString != nil {
		out.ConnectionString = types.StringValue(*details.ConnectionString)
	} else {
		out.ConnectionString = types.StringNull()
	}
	if len(details.Config) > 0 {
		out.Config = jsontypes.NewNormalizedValue(string(details.Config))
	} else {
		out.Config = jsontypes.NewNormalizedNull()
	}
	out.Audit = providerschema.NewAuditObject(ctx, details.Audit)

	// Preserve RequiresReplace / create-only fields from plan/state when present.
	// GET does not return all of these, and optional compute fields would otherwise
	// drift between null (config) and 0 (API zero-value).
	if !base.CloudConfig.IsNull() && !base.CloudConfig.IsUnknown() {
		out.CloudConfig = base.CloudConfig
	} else if details.CloudConfig != nil {
		out.CloudConfig = providerschema.NewCloudConfigObject(ctx, details.CloudConfig)
	}
	if !base.CatalogModelName.IsNull() && !base.CatalogModelName.IsUnknown() {
		out.CatalogModelName = base.CatalogModelName
	}
	if !base.Quantization.IsNull() && !base.Quantization.IsUnknown() {
		out.Quantization = base.Quantization
	}
	if !base.Optimization.IsNull() && !base.Optimization.IsUnknown() {
		out.Optimization = base.Optimization
	}
	if !base.Dimensions.IsNull() && !base.Dimensions.IsUnknown() {
		out.Dimensions = base.Dimensions
	}
	if !base.Guardrails.IsNull() && !base.Guardrails.IsUnknown() {
		out.Guardrails = base.Guardrails
	} else if out.Guardrails.IsUnknown() {
		out.Guardrails = types.ListNull(types.StringType)
	}
	if !base.Jailbreak.IsNull() && !base.Jailbreak.IsUnknown() {
		out.Jailbreak = base.Jailbreak
	} else if out.Jailbreak.IsUnknown() {
		out.Jailbreak = jsontypes.NewNormalizedNull()
	}
	if !base.Caching.IsNull() && !base.Caching.IsUnknown() {
		out.Caching = base.Caching
	} else if out.Caching.IsUnknown() {
		out.Caching = jsontypes.NewNormalizedNull()
	}
	if !base.EnableBatching.IsNull() && !base.EnableBatching.IsUnknown() {
		out.EnableBatching = base.EnableBatching
	}
	if !base.KeywordFiltering.IsNull() && !base.KeywordFiltering.IsUnknown() {
		out.KeywordFiltering = base.KeywordFiltering
	} else if out.KeywordFiltering.IsUnknown() {
		out.KeywordFiltering = types.ListNull(types.StringType)
	}

	return &out
}

func initializeModelComputed(plan providerschema.Model) providerschema.Model {
	plan.Status = types.StringNull()
	plan.ConnectionString = types.StringNull()
	plan.Config = jsontypes.NewNormalizedNull()
	plan.Audit = nullAuditObject()
	return plan
}

func buildCreateModelRequest(ctx context.Context, plan providerschema.Model) (*api.CreateModelRequest, error) {
	cloudCfg, err := cloudConfigFromPlan(ctx, plan.CloudConfig)
	if err != nil {
		return nil, err
	}

	req := &api.CreateModelRequest{
		Name:             plan.Name.ValueString(),
		CatalogModelName: plan.CatalogModelName.ValueString(),
		CloudConfig:      *cloudCfg,
	}

	if !plan.Quantization.IsNull() && !plan.Quantization.IsUnknown() {
		v := plan.Quantization.ValueString()
		req.Quantization = &v
	}
	if !plan.Optimization.IsNull() && !plan.Optimization.IsUnknown() {
		v := plan.Optimization.ValueString()
		req.Optimization = &v
	}
	if !plan.Dimensions.IsNull() && !plan.Dimensions.IsUnknown() {
		v := int(plan.Dimensions.ValueInt64())
		req.Dimensions = &v
	}
	if !plan.Guardrails.IsNull() && !plan.Guardrails.IsUnknown() {
		vals, err := listToStrings(ctx, plan.Guardrails)
		if err != nil {
			return nil, err
		}
		req.Guardrails = vals
	}
	if !plan.Jailbreak.IsNull() && !plan.Jailbreak.IsUnknown() {
		req.Jailbreak = json.RawMessage(plan.Jailbreak.ValueString())
	}
	if !plan.Caching.IsNull() && !plan.Caching.IsUnknown() {
		req.Caching = json.RawMessage(plan.Caching.ValueString())
	}
	if !plan.EnableBatching.IsNull() && !plan.EnableBatching.IsUnknown() {
		v := plan.EnableBatching.ValueBool()
		req.EnableBatching = &v
	}
	if !plan.KeywordFiltering.IsNull() && !plan.KeywordFiltering.IsUnknown() {
		vals, err := listToStrings(ctx, plan.KeywordFiltering)
		if err != nil {
			return nil, err
		}
		req.KeywordFiltering = vals
	}

	return req, nil
}

func buildUpdateModelRequest(ctx context.Context, plan providerschema.Model) (*api.UpdateModelRequest, error) {
	name := plan.Name.ValueString()
	req := &api.UpdateModelRequest{Name: &name}

	if !plan.Guardrails.IsNull() && !plan.Guardrails.IsUnknown() {
		vals, err := listToStrings(ctx, plan.Guardrails)
		if err != nil {
			return nil, err
		}
		req.Guardrails = vals
	}
	if !plan.Jailbreak.IsNull() && !plan.Jailbreak.IsUnknown() {
		req.Jailbreak = json.RawMessage(plan.Jailbreak.ValueString())
	}
	if !plan.Caching.IsNull() && !plan.Caching.IsUnknown() {
		req.Caching = json.RawMessage(plan.Caching.ValueString())
	}
	if !plan.EnableBatching.IsNull() && !plan.EnableBatching.IsUnknown() {
		v := plan.EnableBatching.ValueBool()
		req.EnableBatching = &v
	}
	if !plan.KeywordFiltering.IsNull() && !plan.KeywordFiltering.IsUnknown() {
		vals, err := listToStrings(ctx, plan.KeywordFiltering)
		if err != nil {
			return nil, err
		}
		req.KeywordFiltering = vals
	}

	return req, nil
}

func cloudConfigFromPlan(ctx context.Context, obj types.Object) (*api.CloudConfig, error) {
	if obj.IsNull() || obj.IsUnknown() {
		return nil, fmt.Errorf("cloud_config is required")
	}
	var nested providerschema.CloudConfigModel
	diags := obj.As(ctx, &nested, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("unable to parse cloud_config")
	}
	if nested.Compute.IsNull() || nested.Compute.IsUnknown() {
		return nil, fmt.Errorf("cloud_config.compute is required")
	}
	var compute providerschema.CloudConfigComputeModel
	diags = nested.Compute.As(ctx, &compute, basetypes.ObjectAsOptions{})
	if diags.HasError() {
		return nil, fmt.Errorf("unable to parse cloud_config.compute")
	}

	cfg := &api.CloudConfig{
		Provider: nested.Provider.ValueString(),
		Region:   nested.Region.ValueString(),
		Compute: api.CloudConfigCompute{
			Cpu: int(compute.Cpu.ValueInt64()),
		},
	}
	if !compute.GpuMemory.IsNull() && !compute.GpuMemory.IsUnknown() {
		cfg.Compute.GpuMemory = int(compute.GpuMemory.ValueInt64())
	}
	return cfg, nil
}

func listToStrings(ctx context.Context, list types.List) ([]string, error) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var out []string
	diags := list.ElementsAs(ctx, &out, false)
	if diags.HasError() {
		return nil, fmt.Errorf("unable to convert list to strings")
	}
	return out, nil
}

func stringsToList(ctx context.Context, values []string) types.List {
	if values == nil {
		return types.ListNull(types.StringType)
	}
	l, diags := types.ListValueFrom(ctx, types.StringType, values)
	if diags.HasError() {
		return types.ListNull(types.StringType)
	}
	return l
}

func ifMatchHeadersFromAudit(audit types.Object) map[string]string {
	headers := map[string]string{}
	if audit.IsNull() || audit.IsUnknown() {
		return headers
	}
	attrs := audit.Attributes()
	v, ok := attrs["version"]
	if !ok {
		return headers
	}
	iv, ok := v.(types.Int64)
	if !ok || iv.IsNull() || iv.IsUnknown() {
		return headers
	}
	headers["If-Match"] = strconv.FormatInt(iv.ValueInt64(), 10)
	return headers
}
