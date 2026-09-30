package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/mminichino/couchbase-capella-aidp/internal/api"
	internalerrors "github.com/mminichino/couchbase-capella-aidp/internal/errors"
	providerschema "github.com/mminichino/couchbase-capella-aidp/internal/schema"
)

var (
	_ resource.Resource                = &ModelActivation{}
	_ resource.ResourceWithConfigure   = &ModelActivation{}
	_ resource.ResourceWithImportState = &ModelActivation{}
)

// ModelActivation manages pause/resume for a Capella AI model.
type ModelActivation struct {
	*providerschema.Data
}

// NewModelActivation is a helper function to simplify provider registration.
func NewModelActivation() resource.Resource {
	return &ModelActivation{}
}

func (r *ModelActivation) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_model_activation"
}

func (r *ModelActivation) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = ModelActivationSchema()
}

func (r *ModelActivation) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ModelActivation) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan providerschema.ModelActivation
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateActivationState(plan.ActivationState.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error creating model activation", err.Error())
		return
	}

	orgID := plan.OrganizationID.ValueString()
	modelID := plan.ModelID.ValueString()

	if err := r.applyActivation(ctx, orgID, modelID, plan.ActivationState.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error creating model activation", err.Error())
		return
	}

	status, err := r.getModelStatus(ctx, orgID, modelID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading model after activation", api.ParseError(err))
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/%s", orgID, modelID))
	plan.Status = types.StringValue(status)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

func (r *ModelActivation) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state providerschema.ModelActivation
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	orgID := state.OrganizationID.ValueString()
	modelID := state.ModelID.ValueString()
	if orgID == "" || modelID == "" {
		resp.Diagnostics.AddError("Error reading model activation", "organization_id and model_id are required")
		return
	}

	status, err := r.getModelStatus(ctx, orgID, modelID)
	if err != nil {
		notFound, errString := api.CheckResourceNotFoundError(err)
		if notFound {
			tflog.Info(ctx, "model not found remotely; removing activation from state")
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading model activation", errString)
		return
	}

	state.ID = types.StringValue(fmt.Sprintf("%s/%s", orgID, modelID))
	state.Status = types.StringValue(status)
	// On import, activation_state is unset — derive it heuristically from model status.
	if state.ActivationState.IsNull() || state.ActivationState.IsUnknown() {
		state.ActivationState = types.StringValue(activationStateFromStatus(status))
	}

	diags = resp.State.Set(ctx, state)
	resp.Diagnostics.Append(diags...)
}

func (r *ModelActivation) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan providerschema.ModelActivation
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := validateActivationState(plan.ActivationState.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error updating model activation", err.Error())
		return
	}

	orgID := plan.OrganizationID.ValueString()
	modelID := plan.ModelID.ValueString()

	if err := r.applyActivation(ctx, orgID, modelID, plan.ActivationState.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error updating model activation", err.Error())
		return
	}

	status, err := r.getModelStatus(ctx, orgID, modelID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading model after activation", api.ParseError(err))
		return
	}

	plan.ID = types.StringValue(fmt.Sprintf("%s/%s", orgID, modelID))
	plan.Status = types.StringValue(status)

	diags = resp.State.Set(ctx, plan)
	resp.Diagnostics.Append(diags...)
}

// Delete removes the resource from state only. It does not change the remote model activation state.
func (r *ModelActivation) Delete(ctx context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "removing model activation from state without changing remote model state")
	resp.State.RemoveResource(ctx)
}

func (r *ModelActivation) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := providerschema.SplitImportID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Expected organizationId/modelId: "+err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("organization_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("model_id"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func (r *ModelActivation) applyActivation(ctx context.Context, orgID, modelID, desired string) error {
	status, err := r.getModelStatus(ctx, orgID, modelID)
	if err != nil {
		return fmt.Errorf("%s", api.ParseError(err))
	}
	// Newly deployed models are already healthy ("on"). Capella rejects resume/pause
	// when the model is not in the expected prior state (codes 14014/14015).
	if activationStateFromStatus(status) == desired {
		return nil
	}

	url := fmt.Sprintf("%s/v4/organizations/%s/aiServices/models/%s/activationState", r.HostURL, orgID, modelID)
	var method string
	switch desired {
	case "on":
		method = http.MethodPost
	case "off":
		method = http.MethodDelete
	default:
		return internalerrors.ErrInvalidActivation
	}

	cfg := api.EndpointCfg{
		Url:             url,
		Method:          method,
		SuccessStatus:   http.StatusAccepted,
		SuccessStatuses: []int{http.StatusOK, http.StatusNoContent},
	}
	_, err = r.Client.ExecuteWithRetry(ctx, cfg, nil, r.Token, nil)
	if err != nil {
		return fmt.Errorf("%s", api.ParseError(err))
	}

	return r.waitForActivationState(ctx, orgID, modelID, desired)
}

func (r *ModelActivation) waitForActivationState(ctx context.Context, orgID, modelID, desired string) error {
	const (
		interval = 10 * time.Second
		attempts = 60
	)
	for i := 0; i < attempts; i++ {
		status, err := r.getModelStatus(ctx, orgID, modelID)
		if err != nil {
			return fmt.Errorf("%s", api.ParseError(err))
		}
		s := strings.ToLower(status)
		transitional := strings.Contains(s, "pausing") ||
			strings.Contains(s, "resuming") ||
			strings.Contains(s, "deploying")
		if !transitional && activationStateFromStatus(status) == desired {
			return nil
		}
		tflog.Info(ctx, "waiting for model activation state", map[string]interface{}{
			"model_id":       modelID,
			"desired":        desired,
			"current_status": status,
			"attempt":        i + 1,
		})
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
	return fmt.Errorf("timed out waiting for model %s to reach activation_state %q", modelID, desired)
}

func (r *ModelActivation) getModelStatus(ctx context.Context, orgID, modelID string) (string, error) {
	url := fmt.Sprintf("%s/v4/organizations/%s/aiServices/models/%s", r.HostURL, orgID, modelID)
	cfg := api.EndpointCfg{Url: url, Method: http.MethodGet, SuccessStatus: http.StatusOK}
	response, err := r.Client.ExecuteWithRetry(ctx, cfg, nil, r.Token, nil)
	if err != nil {
		return "", err
	}

	var getResp api.GetModelResponse
	if err := json.Unmarshal(response.Body, &getResp); err != nil {
		return "", err
	}
	if getResp.Model == nil {
		return "", fmt.Errorf("model response missing model payload")
	}
	return getResp.Model.Status, nil
}

func validateActivationState(state string) error {
	switch state {
	case "on", "off":
		return nil
	default:
		return internalerrors.ErrInvalidActivation
	}
}

// activationStateFromStatus maps model status to on/off heuristically for import/read.
// Capella uses "healthy" for an active model and statuses containing "pause" when paused.
func activationStateFromStatus(status string) string {
	s := strings.ToLower(status)
	if strings.Contains(s, "pause") {
		return "off"
	}
	return "on"
}
