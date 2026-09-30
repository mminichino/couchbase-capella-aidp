package resources

import (
	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func ModelSchema() schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Manages a Capella-hosted AI model deployment.",
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
			"name": schema.StringAttribute{
				Required: true,
			},
			"catalog_model_name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cloud_config": schema.SingleNestedAttribute{
				Required: true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"provider": schema.StringAttribute{Required: true},
					"region":   schema.StringAttribute{Required: true},
					"compute": schema.SingleNestedAttribute{
						Required: true,
						Attributes: map[string]schema.Attribute{
							"cpu": schema.Int64Attribute{Required: true},
							"gpu_memory": schema.Int64Attribute{
								Optional:            true,
								MarkdownDescription: "GPU memory in GB (`gpuMemory` in the Capella API).",
							},
						},
					},
				},
			},
			"quantization": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Quantization option. Capella accepts: `int4`, `int8`, `int16`, `fp16`, `fp32`, `fp64`, or `bf16`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"optimization": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optimization profile: `throughput` or `latency`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"dimensions": schema.Int64Attribute{
				Optional: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"guardrails": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"jailbreak": schema.StringAttribute{
				Optional:   true,
				CustomType: jsontypes.NormalizedType{},
			},
			"caching": schema.StringAttribute{
				Optional:   true,
				CustomType: jsontypes.NormalizedType{},
				MarkdownDescription: "JSON caching configuration. For semantic caching, " +
					"`semantic.embeddingModel` must be the deployed embedding model's **name** (not id).",
			},
			"enable_batching": schema.BoolAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"keyword_filtering": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
			"connection_string": schema.StringAttribute{
				Computed: true,
			},
			"config": schema.StringAttribute{
				Computed:   true,
				CustomType: jsontypes.NormalizedType{},
			},
			"audit": computedAuditAttribute(),
		},
	}
}
