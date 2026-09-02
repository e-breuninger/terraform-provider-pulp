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

type PulpDistributionModel struct {
	PulpHref          types.String `tfsdk:"pulp_href"`
	ContentType       types.String `tfsdk:"content_type"`
	PluginName        types.String `tfsdk:"plugin_name"`
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

type pulpDistributionResource struct {
	pulpResource[PulpDistributionModel]
}

func NewPulpDistributionResource() resource.Resource {
	return &pulpDistributionResource{pulpResource[PulpDistributionModel]{
		typeName:    "distribution",
		label:       "Distribution",
		description: "Manages a Pulp Distribution for any content type.",
		collection:  "distributions",
		features:    distributionFeatures,
		fields: variantResourceFields(distributionFeatures,
			field{
				Name: "name", Kind: fieldString, Required: true,
				Description: "A unique name for this Distribution.",
			},
			field{
				Name: "base_path", Kind: fieldString, Required: true,
				Description: "The base_path for this Distribution.",
			},
			field{
				Name: "repository", Kind: fieldString,
				Optional: true, Computed: true, Nullable: true, EmptyIsNull: true,
				Description: "The `pulp_href` of the Repository that should be served at the base_path.",
			},
			field{
				Name: "repository_version", Kind: fieldString,
				Optional: true, Computed: true, Nullable: true, EmptyIsNull: true,
				Description: "The `pulp_href` of the Repository version to serve.",
			},
			field{
				Name: "allow_uploads", Kind: fieldBool,
				Optional: true, Computed: true, Feature: featureAllowUploads,
				Description: "Whether to allow uploads to this Distribution.",
			},
			field{
				Name: "remote", Kind: fieldString,
				Optional: true, Computed: true, EmptyIsNull: true, Feature: featureRemote,
				Description:      "The `pulp_href` of the Remote from which content should be pulled on demand.",
				StringValidators: []validator.String{validators.PulpHrefValidator()},
			},
			field{
				Name: "content_guard", Kind: fieldString,
				Optional: true, Computed: true, EmptyIsNull: true,
				Description:      "The `pulp_href` of the Content Guard to use for this Distribution.",
				StringValidators: []validator.String{validators.PulpHrefValidator()},
			},
			field{
				Name: "namespace", Kind: fieldString,
				Computed: true, ReadOnly: true, EmptyIsNull: true,
				UseStateForUnknown: true, Feature: featureNamespace,
				Description: "The namespace of this Distribution. Only container Distributions have one.",
			},
			field{
				Name: "distributions", Kind: fieldStringList,
				Optional: true, Computed: true, Feature: featureDistributions,
				Description:    "The `pulp_href`s of the Distributions served through this pull-through Distribution.",
				ListValidators: []validator.List{listvalidator.ValueStringsAre(validators.PulpHrefValidator())},
			},
			field{
				Name: "private", Kind: fieldBool,
				Optional: true, Computed: true, Feature: featurePrivate,
				Description: "If true, anonymous users may not pull from this Distribution.",
			},
			labelsField(),
		),
	}}
}
