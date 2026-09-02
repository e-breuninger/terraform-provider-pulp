// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package internal

import (
	"context"
	"fmt"
	"math/big"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Conversions between Pulp's decoded JSON and the terraform-plugin-framework
// value types.

// StrOrNull returns data[key] as a types.String, or null if absent or not a
// string.
func StrOrNull(data map[string]any, key string) types.String {
	if v, ok := data[key].(string); ok {
		return types.StringValue(v)
	}
	return types.StringNull()
}

// StrOrNullNonEmpty also maps "" to null, which is how Pulp reports an unset
// optional field.
func StrOrNullNonEmpty(data map[string]any, key string) types.String {
	if v, ok := data[key].(string); ok && v != "" {
		return types.StringValue(v)
	}
	return types.StringNull()
}

// BoolOrNull returns data[key] as a types.Bool, or null if absent or not a
// bool.
func BoolOrNull(data map[string]any, key string) types.Bool {
	if v, ok := data[key].(bool); ok {
		return types.BoolValue(v)
	}
	return types.BoolNull()
}

// NumberOrNull returns data[key] as a types.Number, or null if absent or not
// a number.
func NumberOrNull(data map[string]any, key string) types.Number {
	if v, ok := data[key].(float64); ok {
		return types.NumberValue(big.NewFloat(v))
	}
	return types.NumberNull()
}

// StringList returns data[key] as a types.List of strings, or a null list.
func StringList(ctx context.Context, data map[string]any, key string) types.List {
	v, ok := data[key].([]any)
	if !ok {
		return types.ListNull(types.StringType)
	}
	strs := make([]string, 0, len(v))
	for _, e := range v {
		if s, ok := e.(string); ok {
			strs = append(strs, s)
		}
	}
	list, diags := types.ListValueFrom(ctx, types.StringType, strs)
	if diags.HasError() {
		return types.ListNull(types.StringType)
	}
	return list
}

// LabelsOrNull converts a response's pulp_labels into a types.Map.
func LabelsOrNull(ctx context.Context, data map[string]any) types.Map {
	v, ok := data["pulp_labels"].(map[string]any)
	if !ok {
		return types.MapNull(types.StringType)
	}
	labels := make(map[string]string, len(v))
	for k, val := range v {
		if s, ok := val.(string); ok {
			labels[k] = s
		}
	}
	m, diags := types.MapValueFrom(ctx, types.StringType, labels)
	if diags.HasError() {
		return types.MapNull(types.StringType)
	}
	return m
}

// ListToStrings converts a types.List to []string, nil when null or unknown.
func ListToStrings(ctx context.Context, l types.List) ([]string, error) {
	if l.IsNull() || l.IsUnknown() {
		return nil, nil
	}
	var out []string
	if diags := l.ElementsAs(ctx, &out, false); diags.HasError() {
		return nil, fmt.Errorf("failed to convert list: %v", diags)
	}
	return out, nil
}

// StringsToList converts a []string into a sorted types.List, so state stays
// stable across reads. The input is not modified.
func StringsToList(in []string) types.List {
	sorted := append([]string(nil), in...)
	sort.Strings(sorted)
	vals := make([]attr.Value, len(sorted))
	for i, s := range sorted {
		vals[i] = types.StringValue(s)
	}
	l, _ := types.ListValue(types.StringType, vals)
	return l
}

// SetStr writes a string attribute into a request body.
//
// Unknown is omitted: PATCH treats an absent key as "leave it alone", while a
// null would be rejected by Pulp's many non-nullable fields. Null means the
// attribute was removed from the config, which only clears the field
// server-side if sent explicitly, so it is sent when nullable.
func SetStr(body map[string]any, key string, s types.String, nullable bool) {
	switch {
	case s.IsUnknown():
	case s.IsNull():
		if nullable {
			body[key] = nil
		}
	default:
		body[key] = s.ValueString()
	}
}

// SetBool is the types.Bool counterpart of SetStr.
func SetBool(body map[string]any, key string, b types.Bool, nullable bool) {
	switch {
	case b.IsUnknown():
	case b.IsNull():
		if nullable {
			body[key] = nil
		}
	default:
		body[key] = b.ValueBool()
	}
}

// SetNumber is the types.Number counterpart of SetStr. Pulp's numeric fields
// are integers, so an exact whole number is sent as one.
func SetNumber(body map[string]any, key string, n types.Number, nullable bool) {
	switch {
	case n.IsUnknown():
	case n.IsNull():
		if nullable {
			body[key] = nil
		}
	default:
		if i, accuracy := n.ValueBigFloat().Int64(); accuracy == big.Exact {
			body[key] = i
			return
		}
		f, _ := n.ValueBigFloat().Float64()
		body[key] = f
	}
}

// SetLabels writes pulp_labels into a request body, omitting null or unknown.
func SetLabels(ctx context.Context, body map[string]any, m types.Map) {
	if m.IsNull() || m.IsUnknown() {
		return
	}
	labels := make(map[string]string, len(m.Elements()))
	m.ElementsAs(ctx, &labels, false)
	body["pulp_labels"] = labels
}

// SetStringList writes a list attribute, omitting null or unknown.
func SetStringList(ctx context.Context, body map[string]any, key string, l types.List) {
	if l.IsNull() || l.IsUnknown() {
		return
	}
	var values []string
	if diags := l.ElementsAs(ctx, &values, false); diags.HasError() {
		return
	}
	body[key] = values
}

// StringSet returns data[key] as a types.Set of strings, or a null set.
func StringSet(ctx context.Context, data map[string]any, key string) types.Set {
	v, ok := data[key].([]any)
	if !ok {
		return types.SetNull(types.StringType)
	}
	strs := make([]string, 0, len(v))
	for _, e := range v {
		if s, ok := e.(string); ok {
			strs = append(strs, s)
		}
	}
	set, diags := types.SetValueFrom(ctx, types.StringType, strs)
	if diags.HasError() {
		return types.SetNull(types.StringType)
	}
	return set
}

// SetStringSet writes a set attribute, omitting null or unknown.
func SetStringSet(ctx context.Context, body map[string]any, key string, s types.Set) {
	if s.IsNull() || s.IsUnknown() {
		return
	}
	var values []string
	if diags := s.ElementsAs(ctx, &values, false); diags.HasError() {
		return
	}
	body[key] = values
}
