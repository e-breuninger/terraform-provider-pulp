// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package modifiers

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

type OrderListModifier struct{}

func (m OrderListModifier) Description(_ context.Context) string {
	return "Suppresses diffs when list elements differ only in order."
}

func (m OrderListModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m OrderListModifier) PlanModifyList(
	ctx context.Context,
	req planmodifier.ListRequest,
	resp *planmodifier.ListResponse,
) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() ||
		req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}

	var stateVals, planVals []string
	resp.Diagnostics.Append(req.StateValue.ElementsAs(ctx, &stateVals, false)...)
	resp.Diagnostics.Append(req.PlanValue.ElementsAs(ctx, &planVals, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if equalFoldSlices(stateVals, planVals) {
		resp.PlanValue = req.StateValue
	}
}

func equalFoldSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	aSet := make(map[string]struct{}, len(a))
	for _, v := range a {
		aSet[strings.ToLower(v)] = struct{}{}
	}

	for _, v := range b {
		if _, ok := aSet[strings.ToLower(v)]; !ok {
			return false
		}
	}

	return true
}
