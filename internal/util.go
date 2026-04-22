// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package internal

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
)

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

func StringList(ctx context.Context, data map[string]any, key string) *types.List {
	if v, ok := data[key].([]any); ok {
		guardElems := make([]types.String, 0, len(v))
		for _, g := range v {
			if s, ok := g.(string); ok {
				guardElems = append(guardElems, types.StringValue(s))
			}
		}
		list, diags := types.ListValueFrom(ctx, types.StringType, guardElems)
		if !diags.HasError() {
			return &list
		}
	}
	return nil
}

// listToStrings converts a types.List of strings to []string, returning nil for null/unknown.
func ListToStrings(ctx context.Context, l types.List) ([]string, error) {
	if l.IsNull() || l.IsUnknown() {
		return nil, nil
	}
	out := make([]string, 0, len(l.Elements()))
	var tmp []string
	diags := l.ElementsAs(ctx, &tmp, false)
	if diags.HasError() {
		return nil, fmt.Errorf("failed to convert list: %v", diags)
	}
	out = append(out, tmp...)
	return out, nil
}

// stringsToList converts a []string into a types.List, sorted for stable state.
func StringsToList(in []string) types.List {
	sort.Strings(in)
	vals := make([]attr.Value, 0, len(in))
	for _, s := range in {
		vals = append(vals, types.StringValue(s))
	}
	l, _ := types.ListValue(types.StringType, vals)
	return l
}

// diff returns (toAdd, toRemove) between desired and current slices.
func Diff(desired, current []string) (toAdd, toRemove []string) {
	d := map[string]struct{}{}
	for _, s := range desired {
		d[s] = struct{}{}
	}
	c := map[string]struct{}{}
	for _, s := range current {
		c[s] = struct{}{}
	}
	for s := range d {
		if _, ok := c[s]; !ok {
			toAdd = append(toAdd, s)
		}
	}
	for s := range c {
		if _, ok := d[s]; !ok {
			toRemove = append(toRemove, s)
		}
	}
	sort.Strings(toAdd)
	sort.Strings(toRemove)
	return
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

func Intersect(want, have []string) []string {
	haveSet := make(map[string]struct{}, len(have))
	for _, s := range have {
		haveSet[s] = struct{}{}
	}
	out := make([]string, 0, len(want))
	for _, s := range want {
		if _, ok := haveSet[s]; ok {
			out = append(out, s)
		}
	}
	return out
}
