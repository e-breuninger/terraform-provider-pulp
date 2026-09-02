// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/e-breuninger/terraform-provider-pulp/internal/validators"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PulpRepositoryModel struct {
	PulpHref    types.String `tfsdk:"pulp_href"`
	ContentType types.String `tfsdk:"content_type"`
	PluginName  types.String `tfsdk:"plugin_name"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Remote      types.String `tfsdk:"remote"`
	PulpLabels  types.Map    `tfsdk:"pulp_labels"`
}

type pulpRepositoryResource struct {
	pulpResource[PulpRepositoryModel]
}

func NewPulpRepositoryResource() resource.Resource {
	return &pulpRepositoryResource{pulpResource[PulpRepositoryModel]{
		typeName:    "repository",
		label:       "Repository",
		description: "Manages a Pulp Repository for any content type.",
		collection:  "repositories",
		features:    repositoryFeatures,
		fields: variantResourceFields(repositoryFeatures,
			field{
				Name: "name", Kind: fieldString, Required: true,
				Description: "A unique name for this Repository.",
			},
			field{
				Name: "description", Kind: fieldString, Required: true,
				Description: "A description for this Repository.",
			},
			field{
				Name: "remote", Kind: fieldString,
				Optional: true, Nullable: true, EmptyIsNull: true,
				Description:      "The `pulp_href` of the Remote this Repository syncs from.",
				StringValidators: []validator.String{validators.PulpHrefValidator()},
			},
			labelsField(),
		),
	}}
}
