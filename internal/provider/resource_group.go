// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PulpGroupModel struct {
	PulpHref types.String `tfsdk:"pulp_href"`
	Name     types.String `tfsdk:"name"`
}

type pulpGroupResource struct {
	pulpResource[PulpGroupModel]
}

func NewPulpGroupResource() resource.Resource {
	return &pulpGroupResource{pulpResource[PulpGroupModel]{
		typeName:    "group",
		label:       "Group",
		description: "Manages a Pulp Group.",
		collection:  "groups",
		fields: []field{
			hrefField(),
			{
				Name: "name", Kind: fieldString, Required: true,
				Description: "A unique name for this Group.",
			},
		},
	}}
}
