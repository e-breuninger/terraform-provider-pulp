// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PulpRemoteModel struct {
	PulpHref      types.String `tfsdk:"pulp_href"`
	ContentType   types.String `tfsdk:"content_type"`
	PluginName    types.String `tfsdk:"plugin_name"`
	Name          types.String `tfsdk:"name"`
	Url           types.String `tfsdk:"url"`
	Policy        types.String `tfsdk:"policy"`
	TlsValidation types.Bool   `tfsdk:"tls_validation"`
	Username      types.String `tfsdk:"username"`
	Password      types.String `tfsdk:"password"`
	PulpLabels    types.Map    `tfsdk:"pulp_labels"`
}

type pulpRemoteResource struct {
	pulpResource[PulpRemoteModel]
}

func NewPulpRemoteResource() resource.Resource {
	return &pulpRemoteResource{pulpResource[PulpRemoteModel]{
		typeName:    "remote",
		label:       "Remote",
		description: "Manages a Pulp Remote for any content type.",
		collection:  "remotes",
		features:    remoteFeatures,
		fields: variantResourceFields(remoteFeatures,
			field{
				Name: "name", Kind: fieldString, Required: true,
				Description: "A unique name for this Remote.",
			},
			field{
				Name: "url", Kind: fieldString, Required: true,
				Description: "The URL of an external content source.",
			},
			field{
				Name: "policy", Kind: fieldString,
				Optional: true, Computed: true, Feature: featurePolicy,
				Description: "Download policy: `immediate`, `on_demand`, or `streamed`.",
				StringValidators: []validator.String{
					stringvalidator.OneOf("immediate", "on_demand", "streamed"),
				},
			},
			field{
				Name: "tls_validation", Kind: fieldBool,
				Optional: true, Computed: true,
				Description: "Whether TLS peer validation must be performed.",
			},
			// Pulp accepts credentials but never reports them back, so they
			// are write-only: hydrating them from a response would clobber
			// the configured value with null.
			field{
				Name: "username", Kind: fieldString,
				Optional: true, WriteOnly: true,
				Description: "Username for authentication when syncing.",
			},
			field{
				Name: "password", Kind: fieldString,
				Optional: true, Sensitive: true, WriteOnly: true,
				Description: "Password for authentication when syncing.",
			},
			labelsField(),
		),
	}}
}
