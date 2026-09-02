// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PulpRoleModel struct {
	PulpHref    types.String `tfsdk:"pulp_href"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Permissions types.Set    `tfsdk:"permissions"`
	Locked      types.Bool   `tfsdk:"locked"`
}

type pulpRoleResource struct {
	pulpResource[PulpRoleModel]
}

func NewPulpRoleResource() resource.Resource {
	return &pulpRoleResource{pulpResource[PulpRoleModel]{
		typeName:    "role",
		label:       "Role",
		description: "Manages a Pulp Role.",
		collection:  "roles",
		fields: []field{
			hrefField(),
			{
				Name: "name", Kind: fieldString, Required: true,
				Description: "A unique name for this Role.",
			},
			{
				Name: "description", Kind: fieldString, Required: true,
				Description: "The description for this Role.",
			},
			{
				Name: "permissions", Kind: fieldStringSet, Required: true,
				Description: "The permissions granted by this Role.",
			},
			{
				Name: "locked", Kind: fieldBool, Computed: true, ReadOnly: true,
				Description: "Whether this Role is built into Pulp and cannot be changed.",
			},
		},
	}}
}
