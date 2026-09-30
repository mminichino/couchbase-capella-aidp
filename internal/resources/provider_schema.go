package resources

import (
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

// ProviderSchema defines the schema for the AI Data Plane provider resource.
func ProviderSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages an AI Data Plane provider integration in Couchbase Capella. " +
			"Supported types: `awsS3`, `openAI`, `awsBedrock`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description: "The ID of the AI Data Plane provider integration.",
			},
			"organization_id": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "The GUID4 ID of the organization.",
			},
			"name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "The name of the provider integration.",
			},
			"type": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				MarkdownDescription: "The type of provider to create. Valid values: `awsS3`, `openAI`, `awsBedrock`.",
			},
			"configuration": schema.StringAttribute{
				Required:    true,
				CustomType:  jsontypes.NormalizedType{},
				Description: "JSON object containing the provider-specific configuration.",
			},
			"audit": computedAuditAttribute(),
		},
	}
}

func computedAuditAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Computed:    true,
		Description: "Couchbase audit data for the resource.",
		Attributes: map[string]schema.Attribute{
			"created_by": schema.StringAttribute{
				Computed:    true,
				Description: "The user who created the resource.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "The RFC3339 timestamp when the resource was created.",
			},
			"modified_by": schema.StringAttribute{
				Computed:    true,
				Description: "The user who last modified the resource.",
			},
			"modified_at": schema.StringAttribute{
				Computed:    true,
				Description: "The RFC3339 timestamp when the resource was last modified.",
			},
			"version": schema.Int64Attribute{
				Computed:    true,
				Description: "The version of the resource used for optimistic concurrency (If-Match).",
			},
		},
	}
}
