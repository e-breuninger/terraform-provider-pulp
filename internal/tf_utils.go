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

// This file holds the conversions between Pulp's decoded JSON
// (map[string]any) and the terraform-plugin-framework value types. Every
// resource hydrates its model from a response map and builds its request
// body from a plan, so these helpers keep that translation in one place
// instead of once per attribute per resource.

// StrOrNull returns data[key] as a types.String, or null if it is absent or
// not a string.
func StrOrNull(data map[string]any, key string) types.String {
	if v, ok := data[key].(string); ok {
		return types.StringValue(v)
	}
	return types.StringNull()
}

// StrOrNullNonEmpty behaves like StrOrNull but also maps an empty string to
// null, matching Pulp's habit of reporting unset optional fields as "".
func StrOrNullNonEmpty(data map[string]any, key string) types.String {
	if v, ok := data[key].(string); ok && v != "" {
		return types.StringValue(v)
	}
	return types.StringNull()
}

// BoolOrNull returns data[key] as a types.Bool, or null if it is absent or
// not a bool.
func BoolOrNull(data map[string]any, key string) types.Bool {
	if v, ok := data[key].(bool); ok {
		return types.BoolValue(v)
	}
	return types.BoolNull()
}

// NumberOrNull returns data[key] as a types.Number, or null if it is absent
// or not a number. encoding/json decodes every JSON number into a float64.
func NumberOrNull(data map[string]any, key string) types.Number {
	if v, ok := data[key].(float64); ok {
		return types.NumberValue(big.NewFloat(v))
	}
	return types.NumberNull()
}

// StringList returns data[key] as a types.List of strings, or a null list if
// it is absent, not a list, or cannot be converted.
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

// LabelsOrNull converts the pulp_labels object of a Pulp response into a
// types.Map of strings.
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

// ListToStrings converts a types.List of strings to []string, returning nil
// for a null or unknown list.
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

// StringsToList converts a []string into a sorted types.List. Sorting keeps
// state stable across reads for collections Pulp returns in arbitrary order.
// The input is not modified.
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

// SetStr writes an Optional (or Optional+Computed) string attribute into a
// Pulp request body.
//
// Unknown always means "omit": the framework leaves an Optional+Computed
// attribute unknown whenever some *other* attribute of the resource changed,
// and Pulp's PATCH treats an absent key as "leave it alone" — which is
// exactly what we want. Sending null instead would break the many Pulp
// fields declared non-nullable ("This field may not be null.").
//
// Null means "the practitioner removed this from the config". That only
// clears the field server-side if we say so explicitly, so it is sent as
// JSON null when nullable is true, and omitted otherwise (Pulp would reject
// the null on a non-nullable field).
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

// SetLabels writes a pulp_labels map into a Pulp request body, omitting it
// when null or unknown.
func SetLabels(ctx context.Context, body map[string]any, m types.Map) {
	if m.IsNull() || m.IsUnknown() {
		return
	}
	labels := make(map[string]string, len(m.Elements()))
	m.ElementsAs(ctx, &labels, false)
	body["pulp_labels"] = labels
}

// SetStringList writes a list-of-strings attribute into a Pulp request body,
// omitting it when null or unknown.
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

// StringSet returns data[key] as a types.Set of strings, or a null set if it
// is absent, not a list, or cannot be converted.
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

// SetStringSet writes a set-of-strings attribute into a Pulp request body,
// omitting it when null or unknown.
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
