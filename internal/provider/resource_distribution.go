// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/e-breuninger/terraform-provider-pulp/internal"
	"github.com/e-breuninger/terraform-provider-pulp/internal/validators"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// pulp_container keeps a pull-through distribution's registry path in a label
// and stores a generated UUID in base_path, so that the parent does not
// overlap the child distributions it creates on pull. See
// https://github.com/pulp/pulp_container/issues/2494.
const (
	containerPullThroughVariant  = "container/pull-through"
	pullThroughDistributionLabel = "pulp_container.pull_through"
)

// isContainerPullThrough reports whether a model addresses the one variant
// that rewrites base_path server-side.
func isContainerPullThrough(model *PulpDistributionModel) bool {
	return variantKey(model.ContentType.ValueString(), model.PluginName.ValueString()) ==
		containerPullThroughVariant
}

type PulpDistributionModel struct {
	PulpHref          types.String `tfsdk:"pulp_href"`
	Prn               types.String `tfsdk:"prn"`
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

		// Report the registry path the configuration asked for rather than
		// the UUID Pulp put in base_path, and keep the bookkeeping label out
		// of pulp_labels so it does not read as drift.
		afterHydrate: func(ctx context.Context, data map[string]any, model *PulpDistributionModel) {
			if !isContainerPullThrough(model) {
				return
			}
			labels, ok := data["pulp_labels"].(map[string]any)
			if !ok {
				return
			}
			path, ok := labels[pullThroughDistributionLabel].(string)
			if !ok || path == "" {
				return
			}
			model.BasePath = types.StringValue(path)
			delete(labels, pullThroughDistributionLabel)
			model.PulpLabels = internal.LabelsOrNull(ctx, data)
		},

		// Pulp refuses to update a marked pull-through distribution's
		// base_path. Drop the attribute while the configuration still asks
		// for the same path, so unrelated changes apply; send it when it
		// changed, so Pulp's own validation error reaches the user.
		beforeUpdate: func(_ context.Context, plan, state *PulpDistributionModel, body map[string]any) {
			if isContainerPullThrough(plan) && plan.BasePath.Equal(state.BasePath) {
				delete(body, "base_path")
			}
		},
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
