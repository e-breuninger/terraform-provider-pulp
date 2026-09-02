// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"math/big"
	"regexp"

	"github.com/e-breuninger/terraform-provider-pulp/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PulpUserRoleModel struct {
	PulpHref         types.String `tfsdk:"pulp_href"`
	UserID           types.Number `tfsdk:"user_id"`
	Role             types.String `tfsdk:"role"`
	ContentObject    types.String `tfsdk:"content_object"`
	ContentObjectPrn types.String `tfsdk:"content_object_prn"`
	Domain           types.String `tfsdk:"domain"`
}

type pulpUserRoleResource struct {
	pulpResource[PulpUserRoleModel]
}

// userRoleHrefRegex pulls the user ID back out of the href Pulp returns,
// e.g. /pulp/api/v3/users/3/roles/<uuid>/.
var userRoleHrefRegex = regexp.MustCompile(`/users/(\d+)/roles/`)

func NewPulpUserRoleResource() resource.Resource {
	return &pulpUserRoleResource{pulpResource[PulpUserRoleModel]{
		typeName: "user_role",
		label:    "UserRole",
		description: "Assigns a Pulp Role to a User. Pulp has no way to modify an existing " +
			"assignment, so every change replaces it.",
		collection: "users",

		// A UserRole is nested under the user it belongs to rather than
		// living in a collection of its own.
		resourcePath: func(model *PulpUserRoleModel) string {
			return client.BuildResourcePath("users", userIDPath(model.UserID), "roles")
		},

		// user_id is part of the URL, not the body, and Pulp does not report
		// it as a field — but its href encodes it, which is what makes import
		// work.
		afterHydrate: func(_ context.Context, data map[string]any, model *PulpUserRoleModel) {
			href, _ := data["pulp_href"].(string)
			matches := userRoleHrefRegex.FindStringSubmatch(href)
			if len(matches) != 2 {
				return
			}
			if n, ok := new(big.Float).SetString(matches[1]); ok {
				model.UserID = types.NumberValue(n)
			}
		},

		// Pulp offers no PATCH for a role assignment: the only way to change
		// one is to drop it and add another. Requiring replacement on every
		// attribute lets Terraform do exactly that, in the right order and
		// visibly in the plan, instead of the resource silently deleting and
		// re-creating itself during an update.
		fields: []field{
			hrefField(),
			{
				Name: "user_id", Kind: fieldNumber,
				Required: true, RequiresReplace: true, Local: true,
				Description: "The ID of the User that gets this Role.",
			},
			{
				Name: "role", Kind: fieldString, Required: true, RequiresReplace: true,
				Description: "The Role to assign to the User.",
			},
			{
				Name: "content_object", Kind: fieldString,
				Optional: true, RequiresReplace: true, Nullable: true, EmptyIsNull: true,
				Description: "The `pulp_href` of the object this Role applies to. Leave unset to grant the Role at domain or model level.",
			},
			{
				Name: "content_object_prn", Kind: fieldString,
				Optional: true, RequiresReplace: true, Nullable: true, EmptyIsNull: true,
				Description: "The PRN of the object this Role applies to. Leave unset to grant the Role at domain or model level.",
			},
			{
				Name: "domain", Kind: fieldString,
				Optional: true, RequiresReplace: true,
				Description: "The domain this Role applies to. Mutually exclusive with `content_object`.",
			},
		},
	}}
}

// userIDPath renders a user ID for use in a URL path.
func userIDPath(id types.Number) string {
	if id.IsNull() || id.IsUnknown() {
		return ""
	}
	return id.ValueBigFloat().Text('f', -1)
}
