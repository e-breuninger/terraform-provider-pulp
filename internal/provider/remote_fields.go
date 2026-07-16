// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	internal "github.com/e-breuninger/terraform-provider-pulp/internal"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// remoteFields holds the terraform attribute values shared between the
// pulp_remote resource and each item in the pulp_remotes data source's
// "remotes" list. Both hydrate from the same Pulp API response shape
// (minus content_type and plugin_name, which are request inputs rather
// than server fields, and minus password, which Pulp never returns).
//
// Note: Username is also write-only in Pulp's API and is never present in
// GET/list responses, so hydrateRemoteFields always resolves it to null
// here. That's fine for the read-only data sources (there's simply nothing
// to report), but the pulp_remote *resource* must not use this struct's
// Username value to overwrite its own state/plan value - see the comment
// on hydrateRemoteModel in resource_remote.go.
type remoteFields struct {
	PulpHref      types.String `tfsdk:"pulp_href"`
	Name          types.String `tfsdk:"name"`
	Url           types.String `tfsdk:"url"`
	Policy        types.String `tfsdk:"policy"`
	TlsValidation types.Bool   `tfsdk:"tls_validation"`
	Username      types.String `tfsdk:"username"`
	PulpLabels    types.Map    `tfsdk:"pulp_labels"`
}

// remoteFieldsAttrTypes mirrors remoteFields for use with
// types.ObjectValueFrom when building nested remote objects.
var remoteFieldsAttrTypes = map[string]attr.Type{
	"pulp_href":      types.StringType,
	"name":           types.StringType,
	"url":            types.StringType,
	"policy":         types.StringType,
	"tls_validation": types.BoolType,
	"username":       types.StringType,
	"pulp_labels":    types.MapType{ElemType: types.StringType},
}

// hydrateRemoteFields extracts the shared remote fields from a Pulp API
// response map.
func hydrateRemoteFields(ctx context.Context, data map[string]any) remoteFields {
	return remoteFields{
		PulpHref:      internal.StrOrNull(data, "pulp_href"),
		Name:          internal.StrOrNull(data, "name"),
		Url:           internal.StrOrNull(data, "url"),
		Policy:        internal.StrOrNull(data, "policy"),
		TlsValidation: internal.BoolOrNull(data, "tls_validation"),
		Username:      internal.StrOrNull(data, "username"),
		PulpLabels:    internal.LabelsOrNull(ctx, data),
	}
}
