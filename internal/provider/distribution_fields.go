// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	internal "github.com/e-breuninger/terraform-provider-pulp/internal"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// distributionFields holds the terraform attribute values shared between the
// pulp_distribution resource, the pulp_distribution data source, and each
// item in the pulp_distributions data source's "distributions" list. All
// three hydrate from the same Pulp API response shape (minus content_type
// and plugin_name, which are request inputs rather than server fields).
type distributionFields struct {
	PulpHref          types.String `tfsdk:"pulp_href"`
	Name              types.String `tfsdk:"name"`
	BasePath          types.String `tfsdk:"base_path"`
	Repository        types.String `tfsdk:"repository"`
	RepositoryVersion types.String `tfsdk:"repository_version"`
	AllowUploads      types.Bool   `tfsdk:"allow_uploads"`
	Remote            types.String `tfsdk:"remote"`
	ContentGuard      types.String `tfsdk:"content_guard"`
	Namespace         types.String `tfsdk:"namespace"`
	Private           types.Bool   `tfsdk:"private"`
	Distributions     types.List   `tfsdk:"distributions"`
	PulpLabels        types.Map    `tfsdk:"pulp_labels"`
}

// distributionFieldsAttrTypes mirrors distributionFields for use with
// types.ObjectValueFrom when building nested distribution objects.
var distributionFieldsAttrTypes = map[string]attr.Type{
	"pulp_href":          types.StringType,
	"name":               types.StringType,
	"base_path":          types.StringType,
	"repository":         types.StringType,
	"repository_version": types.StringType,
	"allow_uploads":      types.BoolType,
	"remote":             types.StringType,
	"content_guard":      types.StringType,
	"namespace":          types.StringType,
	"private":            types.BoolType,
	"distributions":      types.ListType{ElemType: types.StringType},
	"pulp_labels":        types.MapType{ElemType: types.StringType},
}

// hydrateDistributionFields extracts the shared distribution fields from a
// Pulp API response map.
func hydrateDistributionFields(ctx context.Context, data map[string]any) distributionFields {
	f := distributionFields{
		PulpHref:          internal.StrOrNull(data, "pulp_href"),
		Name:              internal.StrOrNull(data, "name"),
		BasePath:          internal.StrOrNull(data, "base_path"),
		Repository:        internal.StrOrNullNonEmpty(data, "repository"),
		RepositoryVersion: internal.StrOrNullNonEmpty(data, "repository_version"),
		AllowUploads:      internal.BoolOrNull(data, "allow_uploads"),
		Remote:            internal.StrOrNullNonEmpty(data, "remote"),
		ContentGuard:      internal.StrOrNullNonEmpty(data, "content_guard"),
		Namespace:         internal.StrOrNullNonEmpty(data, "namespace"),
		Private:           internal.BoolOrNull(data, "private"),
		PulpLabels:        internal.LabelsOrNull(ctx, data),
	}

	if _, ok := data["distributions"].([]any); ok {
		f.Distributions = *internal.StringList(ctx, data, "distributions")
	} else {
		f.Distributions = types.ListNull(types.StringType)
	}

	return f
}
