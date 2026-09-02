// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PulpUserModel struct {
	PulpHref  types.String `tfsdk:"pulp_href"`
	ID        types.Number `tfsdk:"id"`
	Username  types.String `tfsdk:"username"`
	Password  types.String `tfsdk:"password"`
	FirstName types.String `tfsdk:"first_name"`
	LastName  types.String `tfsdk:"last_name"`
	Email     types.String `tfsdk:"email"`
	IsStaff   types.Bool   `tfsdk:"is_staff"`
	IsActive  types.Bool   `tfsdk:"is_active"`
}

type pulpUserResource struct {
	pulpResource[PulpUserModel]
}

func NewPulpUserResource() resource.Resource {
	return &pulpUserResource{pulpResource[PulpUserModel]{
		typeName:    "user",
		label:       "User",
		description: "Manages a Pulp User.",
		collection:  "users",
		fields: []field{
			hrefField(),
			{
				Name: "id", Kind: fieldNumber,
				Computed: true, ReadOnly: true, UseStateForUnknown: true,
				Description: "The Pulp user ID.",
			},
			{
				Name: "username", Kind: fieldString, Required: true,
				Description: "A unique username for this User. 150 characters or fewer. Letters, digits and `@.+-_` only.",
				StringValidators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z0-9@.+_-]{1,150}$`),
						"must be 150 characters or fewer and contain only letters, digits, and @.+-_"),
				},
			},
			{
				// Pulp stores only the hash and never reports the password
				// back, so it is write-only. It is nullable so that removing
				// it from the config clears it rather than silently leaving
				// the old one in place.
				Name: "password", Kind: fieldString,
				Optional: true, Sensitive: true, WriteOnly: true, Nullable: true,
				Description: "The password for this User. Pulp allows empty passwords but they are not recommended.",
			},
			{
				Name: "first_name", Kind: fieldString, Optional: true, Computed: true,
				Description: "The first name of this User.",
			},
			{
				Name: "last_name", Kind: fieldString, Optional: true, Computed: true,
				Description: "The last name of this User.",
			},
			{
				Name: "email", Kind: fieldString, Optional: true, Computed: true,
				Description: "The email address of this User.",
			},
			{
				Name: "is_staff", Kind: fieldBool, Optional: true, Computed: true,
				Description: "Whether this User can log into the admin site.",
			},
			{
				Name: "is_active", Kind: fieldBool, Optional: true, Computed: true,
				Description: "Whether this User account is active.",
			},
		},
	}}
}
