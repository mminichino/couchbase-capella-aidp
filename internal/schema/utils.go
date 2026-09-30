package schema

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// StringsToSet converts a string slice to a Terraform set of strings.
func StringsToSet(values []string) types.Set {
	if values == nil {
		return types.SetNull(types.StringType)
	}
	elems := make([]types.String, len(values))
	for i, v := range values {
		elems[i] = types.StringValue(v)
	}
	set, diags := types.SetValueFrom(context.Background(), types.StringType, elems)
	if diags.HasError() {
		return types.SetNull(types.StringType)
	}
	return set
}

// SetToStrings converts a Terraform set of strings to a string slice.
func SetToStrings(ctx context.Context, set types.Set) ([]string, error) {
	if set.IsNull() || set.IsUnknown() {
		return nil, nil
	}
	var elems []types.String
	diags := set.ElementsAs(ctx, &elems, false)
	if diags.HasError() {
		return nil, fmt.Errorf("unable to convert set to strings")
	}
	out := make([]string, len(elems))
	for i, e := range elems {
		out[i] = e.ValueString()
	}
	return out, nil
}

// BaseStringsToStrings extracts string values.
func BaseStringsToStrings(stringValues []basetypes.StringValue) []string {
	result := make([]string, len(stringValues))
	for i, sv := range stringValues {
		result[i] = sv.ValueString()
	}
	return result
}

// SplitImportID splits an import ID of the form id1/id2/... into parts.
func SplitImportID(id string, expected int) ([]string, error) {
	parts := strings.Split(id, "/")
	if len(parts) != expected {
		return nil, fmt.Errorf("expected import id with %d parts separated by '/', got %d", expected, len(parts))
	}
	for _, p := range parts {
		if p == "" {
			return nil, fmt.Errorf("import id parts must not be empty")
		}
	}
	return parts, nil
}
