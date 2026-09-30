package datasources

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

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
				Description: "The version of the resource.",
			},
		},
	}
}
