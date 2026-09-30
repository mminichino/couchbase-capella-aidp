package resources

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

func ModelActivationSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages pause/resume (activation state) for a Capella AI model. " +
			"Destroying this resource only removes it from Terraform state and does not change the model activation state.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"organization_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"model_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"activation_state": schema.StringAttribute{
				Required: true,
				MarkdownDescription: `Desired activation state: "on" (resume) or "off" (pause). ` +
					`Applying the same state the model already has is a no-op (newly deployed models are already "on"/healthy).`,
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current model status after applying the activation change.",
			},
		},
	}
}
