package schema

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mminichino/couchbase-capella-aidp/internal/api"
)

// AuditAttrTypes are the attribute types for audit nested objects.
func AuditAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"created_by":  types.StringType,
		"created_at":  types.StringType,
		"modified_by": types.StringType,
		"modified_at": types.StringType,
		"version":     types.Int64Type,
	}
}

// NewAuditObject builds an audit object from API audit data.
func NewAuditObject(_ context.Context, audit *api.CouchbaseAuditData) types.Object {
	if audit == nil {
		return types.ObjectNull(AuditAttrTypes())
	}
	obj, _ := types.ObjectValue(AuditAttrTypes(), map[string]attr.Value{
		"created_by":  types.StringValue(audit.CreatedBy),
		"created_at":  types.StringValue(audit.CreatedAt),
		"modified_by": types.StringValue(audit.ModifiedBy),
		"modified_at": types.StringValue(audit.ModifiedAt),
		"version":     types.Int64Value(int64(audit.Version)),
	})
	return obj
}

// CloudConfigAttrTypes are attribute types for cloud_config.
func CloudConfigAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"provider": types.StringType,
		"region":   types.StringType,
		"compute":  types.ObjectType{AttrTypes: CloudConfigComputeAttrTypes()},
	}
}

// CloudConfigComputeAttrTypes are attribute types for compute.
func CloudConfigComputeAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"cpu":        types.Int64Type,
		"gpu_memory": types.Int64Type,
	}
}

// NewCloudConfigObject builds a cloud_config object from API data.
func NewCloudConfigObject(_ context.Context, cfg *api.CloudConfig) types.Object {
	if cfg == nil {
		return types.ObjectNull(CloudConfigAttrTypes())
	}
	compute, _ := types.ObjectValue(CloudConfigComputeAttrTypes(), map[string]attr.Value{
		"cpu":        types.Int64Value(int64(cfg.Compute.Cpu)),
		"gpu_memory": types.Int64Value(int64(cfg.Compute.GpuMemory)),
	})
	obj, _ := types.ObjectValue(CloudConfigAttrTypes(), map[string]attr.Value{
		"provider": types.StringValue(cfg.Provider),
		"region":   types.StringValue(cfg.Region),
		"compute":  compute,
	})
	return obj
}
