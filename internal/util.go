// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package internal

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
