// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package internal

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
)

// ImportState is a helper function to import a resource by its pulp_href,
// parsing content_type and plugin_name for use in the resource.
func ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) []string {
	pulpHref := req.ID

	// Example href: /pulp/api/v3/repositories/npm/npm/<uuid>/
	// Parse content_type and plugin_name from the href
	parts := strings.Split(strings.Trim(pulpHref, "/"), "/")
	// parts: ["pulp", "api", "v3", "repositories", "<content_type>", "<plugin_name>", "<uuid>"]
	if len(parts) < 2 {
		resp.Diagnostics.AddError("Invalid pulp_href",
			fmt.Sprintf("Could not parse content_type and plugin_name from pulp_href '%s', got %d parts: %v", pulpHref, len(parts), parts))
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("pulp_href"), req.ID)...)
	return parts
}

func CompositeID(cgHref, role string) string {
	return fmt.Sprintf("%s|%s", cgHref, role)
}

func SplitCompositeID(id string) (cgHref, role string, err error) {
	parts := strings.SplitN(id, "|", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid composite ID %q, expected `<contentguard_href>|<role>`", id)
	}
	return parts[0], parts[1], nil
}

func RandomSuffix() string {
	return acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
}

// Types
func StringList(ctx context.Context, data map[string]any, key string) *types.List {
	v, ok := data[key].([]any)
	if !ok {
		return nil
	}
	strs := make([]string, 0, len(v))
	for _, g := range v {
		if s, ok := g.(string); ok {
			strs = append(strs, s)
		}
	}
	list, diags := types.ListValueFrom(ctx, types.StringType, strs)
	if diags.HasError() {
		return nil
	}
	return &list
}

// ListToStrings converts a types.List of strings to []string, returning nil for null/unknown.
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

// StringsToList converts a []string into a sorted types.List for stable state.
// Does not mutate the input.
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

func StrOrNull(data map[string]any, key string) types.String {
	if v, ok := data[key].(string); ok {
		return types.StringValue(v)
	}
	return types.StringNull()
}

func IntOrNull(data map[string]any, key string) types.Int64 {
	if v, ok := data[key].(float64); ok {
		return types.Int64Value(int64(v))
	}
	return types.Int64Null()
}

func BoolOrNull(data map[string]any, key string) types.Bool {
	if v, ok := data[key].(bool); ok {
		return types.BoolValue(v)
	}
	return types.BoolNull()
}

// SetStrField sets body[key] from an Optional (or Optional+Computed) string
// attribute, for use when building a Pulp request body.
//
// If s is Unknown (e.g. an Optional+Computed attribute the framework hasn't
// resolved yet because some other attribute on the resource changed), the
// key is omitted entirely rather than sent as JSON null: Pulp's PATCH
// endpoints treat a missing field as "leave it alone", but a great many
// Pulp fields (e.g. Remote.policy, Remote.tls_validation) are declared
// non-nullable and reject an explicit null with "This field may not be
// null." Naively converting Unknown to nil (as a prior version of this
// helper did) would intermittently send null for such fields on any
// update that touches an unrelated attribute.
//
// If s is explicitly Null, the key is set to JSON null only when nullable
// is true (i.e. the target Pulp field actually accepts null, typically used
// to clear it); otherwise the key is omitted, since Pulp would reject the
// null the same way.
func SetStrField(body map[string]any, key string, s types.String, nullable bool) {
	if s.IsUnknown() {
		return
	}
	if s.IsNull() {
		if nullable {
			body[key] = nil
		}
		return
	}
	body[key] = s.ValueString()
}

// SetBoolField is the types.Bool counterpart of SetStrField.
func SetBoolField(body map[string]any, key string, b types.Bool, nullable bool) {
	if b.IsUnknown() {
		return
	}
	if b.IsNull() {
		if nullable {
			body[key] = nil
		}
		return
	}
	body[key] = b.ValueBool()
}

// SetIntField is the types.Int64 counterpart of SetStrField.
func SetIntField(body map[string]any, key string, i types.Int64, nullable bool) {
	if i.IsUnknown() {
		return
	}
	if i.IsNull() {
		if nullable {
			body[key] = nil
		}
		return
	}
	body[key] = i.ValueInt64()
}

func NumberOrNull(data map[string]any, key string) types.Number {
	if v, ok := data[key].(float64); ok {
		return types.NumberValue(big.NewFloat(v))
	}
	return types.NumberNull()
}

// StrOrNullNonEmpty behaves like StrOrNull but treats an empty string as null,
// matching Pulp's convention of returning "" for unset optional fields.
func StrOrNullNonEmpty(data map[string]any, key string) types.String {
	if v, ok := data[key].(string); ok && v != "" {
		return types.StringValue(v)
	}
	return types.StringNull()
}

// LabelsOrNull converts the pulp_labels map from a Pulp API response into a types.Map.
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
	m, _ := types.MapValueFrom(ctx, types.StringType, labels)
	return m
}

// LabelsToMap converts a types.Map of pulp_labels into a map[string]string,
// returning nil if the map is null or unknown.
func LabelsToMap(ctx context.Context, m types.Map) map[string]string {
	if m.IsNull() || m.IsUnknown() {
		return nil
	}
	labels := make(map[string]string)
	m.ElementsAs(ctx, &labels, false)
	return labels
}
