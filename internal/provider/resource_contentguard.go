// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/e-breuninger/terraform-provider-pulp/internal/validators"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PulpContentGuardModel struct {
	PulpHref    types.String `tfsdk:"pulp_href"`
	ContentType types.String `tfsdk:"content_type"`
	PluginName  types.String `tfsdk:"plugin_name"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`

	// Contentguards: X509 and Rhsm exclusive
	CaCertificate types.String `tfsdk:"ca_certificate"`

	// Contentguards: Composite exclusive
	Guards types.List `tfsdk:"guards"`

	// Contentguards: Header exclusive
	HeaderName  types.String `tfsdk:"header_name"`
	HeaderValue types.String `tfsdk:"header_value"`
	JqFilter    types.String `tfsdk:"jq_filter"`

	// Contentguards: Rbac exclusive. Pulp computes these from the role
	// assignments made through pulp_object_role; they are never written here.
	Users  types.List `tfsdk:"users"`
	Groups types.List `tfsdk:"groups"`
}

type pulpContentGuardResource struct {
	pulpResource[PulpContentGuardModel]
}

func NewPulpContentGuardResource() resource.Resource {
	return &pulpContentGuardResource{pulpResource[PulpContentGuardModel]{
		typeName:    "contentguard",
		label:       "ContentGuard",
		description: "Manages a Pulp ContentGuard.",
		collection:  "contentguards",
		features:    contentGuardFeatures,
		fields: variantResourceFields(contentGuardFeatures,
			field{
				Name: "name", Kind: fieldString, Required: true,
				Description: "A unique name for this ContentGuard.",
			},
			field{
				Name: "description", Kind: fieldString,
				Optional: true, Computed: true, Nullable: true,
				Description: "A description for this ContentGuard.",
			},
			field{
				Name: "ca_certificate", Kind: fieldString,
				Optional: true, Feature: featureCaCertificate,
				Description: "The CA certificate client certificates are validated against.",
			},
			field{
				Name: "guards", Kind: fieldStringList,
				Optional: true, Feature: featureGuards,
				Description:    "The `pulp_href`s of the ContentGuards this composite ContentGuard combines.",
				ListValidators: []validator.List{listvalidator.ValueStringsAre(validators.PulpHrefValidator())},
			},
			field{
				Name: "header_name", Kind: fieldString,
				Optional: true, Feature: featureHeaderName,
				Description: "The name of the header to check.",
			},
			field{
				Name: "header_value", Kind: fieldString,
				Optional: true, Feature: featureHeaderValue,
				Description: "The value the header must carry.",
			},
			field{
				Name: "jq_filter", Kind: fieldString,
				Optional: true, Feature: featureJqFilter,
				Description: "A jq filter applied to the decoded header value.",
			},
			field{
				Name: "users", Kind: fieldObjectList,
				Computed: true, ReadOnly: true,
				Description: "The users granted role-based access. Only rbac ContentGuards report these.",
				Nested: []field{
					{Name: "username", Kind: fieldString, Computed: true},
					{Name: "pulp_href", Kind: fieldString, Computed: true},
					{Name: "prn", Kind: fieldString, Computed: true},
				},
			},
			field{
				Name: "groups", Kind: fieldObjectList,
				Computed: true, ReadOnly: true,
				Description: "The groups granted role-based access. Only rbac ContentGuards report these.",
				Nested: []field{
					{Name: "name", Kind: fieldString, Computed: true},
					{Name: "pulp_href", Kind: fieldString, Computed: true},
					{Name: "prn", Kind: fieldString, Computed: true},
					{Name: "id", Kind: fieldNumber, Computed: true},
				},
			},
		),
	}}
}
