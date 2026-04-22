// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/e-breuninger/terraform-provider-pulp/internal"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// TestAccPulpObjectRoleResource_rbacContentGuard exercises the full lifecycle of a
// pulp_object_role against an RBAC content guard, which is the canonical use case:
//
//	pulp content-guard rbac create --name rbac-guard
//	pulp content-guard rbac assign --name rbac-guard --user alice --user bob --group file-buddies
//	pulp file distribution update --name foo --content-guard core:rbac:rbac-guard
//
// The test covers:
//  1. Create with an initial set of users/groups
//  2. Update that adds a user, removes a user, and swaps the group
//  3. Import via `<pulp_href>|<role>`
//  4. Implicit destroy at the end of the TestCase
func TestAccPulpObjectRoleResource_rbacContentGuard(t *testing.T) {
	suffix := internal.RandomSuffix()
	cgName := fmt.Sprintf("tf-acc-cg-%s", suffix)
	distName := fmt.Sprintf("tf-acc-dist-%s", suffix)
	basePath := fmt.Sprintf("tf-acc-%s", suffix)
	userAlice := fmt.Sprintf("tf-acc-alice-%s", suffix)
	userBob := fmt.Sprintf("tf-acc-bob-%s", suffix)
	userCarol := fmt.Sprintf("tf-acc-carol-%s", suffix)
	groupBuddies := fmt.Sprintf("tf-acc-file-buddies-%s", suffix)
	groupFriends := fmt.Sprintf("tf-acc-file-friends-%s", suffix)

	// core.rbaccontentguard_downloader is the role Pulp ships for read access to
	// content protected by an RBAC content guard.
	const role = "core.rbaccontentguard_downloader"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create — alice + bob as users, file-buddies as group.
			{
				Config: testAccPulpObjectRoleConfig_rbac(
					cgName, distName, basePath,
					userAlice, userBob, userCarol,
					groupBuddies, groupFriends,
					role,
					[]string{"admin", userAlice, userBob},
					[]string{groupBuddies},
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("pulp_object_role.guard", "pulp_href"),
					resource.TestCheckResourceAttr("pulp_object_role.guard", "role", role),
					resource.TestCheckResourceAttr("pulp_object_role.guard", "users.#", "3"),
					resource.TestCheckTypeSetElemAttr("pulp_object_role.guard", "users.*", userAlice),
					resource.TestCheckTypeSetElemAttr("pulp_object_role.guard", "users.*", userBob),
					resource.TestCheckResourceAttr("pulp_object_role.guard", "groups.#", "1"),
					resource.TestCheckTypeSetElemAttr("pulp_object_role.guard", "groups.*", groupBuddies),
					// Sanity: the distribution is wired to the guard.
					resource.TestCheckResourceAttrPair(
						"pulp_distribution.dist", "content_guard",
						"pulp_contentguard.guard", "pulp_href",
					),
				),
			},
			// Step 2: Import — the composite ID format is `<pulp_href>|<role>`.
			{
				ResourceName:  "pulp_object_role.guard",
				ImportState:   true,
				ImportStateId: "pulp_object_role.pulp_href",
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["pulp_object_role.guard"]
					if !ok {
						return "", fmt.Errorf("resource pulp_object_role.guard not found in state")
					}
					href := rs.Primary.Attributes["pulp_href"]
					r := rs.Primary.Attributes["role"]
					if href == "" || r == "" {
						return "", fmt.Errorf("missing pulp_href or role in state: %+v", rs.Primary.Attributes)
					}
					return href + "|" + r, nil
				},
			},
			// Step 3: Update — drop bob, add carol; swap group from buddies to friends.
			{
				Config: testAccPulpObjectRoleConfig_rbac(
					cgName, distName, basePath,
					userAlice, userBob, userCarol,
					groupBuddies, groupFriends,
					role,
					[]string{userAlice, userCarol},
					[]string{groupFriends},
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pulp_object_role.guard", "users.#", "2"),
					resource.TestCheckTypeSetElemAttr("pulp_object_role.guard", "users.*", userAlice),
					resource.TestCheckTypeSetElemAttr("pulp_object_role.guard", "users.*", userCarol),
					resource.TestCheckNoResourceAttr("pulp_object_role.guard", "users.*"),
					resource.TestCheckResourceAttr("pulp_object_role.guard", "groups.#", "1"),
					resource.TestCheckTypeSetElemAttr("pulp_object_role.guard", "groups.*", groupFriends),
					resource.TestCheckNoResourceAttr("pulp_object_role.guard", "groups.*"),
				),
			},
		},
	})
}

// TestAccPulpObjectRoleResource_usersOnly verifies that the resource works when only
// users are provided (groups omitted), since both attributes are Optional+Computed.
func TestAccPulpObjectRoleResource_usersOnly(t *testing.T) {
	suffix := internal.RandomSuffix()
	cgName := fmt.Sprintf("tf-acc-cg-%s", suffix)
	userAlice := fmt.Sprintf("tf-acc-alice-%s", suffix)
	const role = "core.rbaccontentguard_downloader"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + fmt.Sprintf(`
resource "pulp_contentguard" "guard" {
  name = %[1]q
  content_type = "core"
	plugin_name = "rbac"
}

resource "pulp_user" "alice" {
  username = %[2]q
}

resource "pulp_object_role" "guard" {
  pulp_href = pulp_contentguard.guard.pulp_href
  role      = %[3]q
  users     = ["admin", pulp_user.alice.username]
}
`, cgName, userAlice, role),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pulp_object_role.guard", "users.#", "2"),
					resource.TestCheckTypeSetElemAttr("pulp_object_role.guard", "users.*", userAlice),
					resource.TestCheckResourceAttr("pulp_object_role.guard", "groups.#", "0"),
				),
			},
		},
	})
}

// testAccPulpObjectRoleConfig_rbac renders a complete HCL config with a content guard,
// a distribution bound to it, three users, two groups, and the role assignment under test.
// The `users` / `groups` arguments control which subset is assigned by the role.
func testAccPulpObjectRoleConfig_rbac(
	cgName, distName, basePath,
	userAlice, userBob, userCarol,
	groupBuddies, groupFriends,
	role string,
	users, groups []string,
) string {
	return providerConfig + fmt.Sprintf(`
resource "pulp_contentguard" "guard" {
  name = %[1]q
	content_type = "core"
  plugin_name = "rbac"
}

resource "pulp_distribution" "dist" {
  name          = %[2]q
  base_path     = %[3]q
  plugin_name   = "file"
	content_type  = "file"
  content_guard = pulp_contentguard.guard.pulp_href
}

resource "pulp_user" "alice" {
  username = %[4]q
}

resource "pulp_user" "bob" {
  username = %[5]q
}

resource "pulp_user" "carol" {
  username = %[6]q
}

resource "pulp_group" "buddies" {
  name = %[7]q
}

resource "pulp_group" "friends" {
  name = %[8]q
}

# Ensure the referenced principals exist before the role is assigned. Terraform will
# infer this from the attribute references below, but listing the groups in depends_on
# makes the ordering explicit for the groups that aren't otherwise referenced.
resource "pulp_object_role" "guard" {
  pulp_href = pulp_contentguard.guard.pulp_href
  role      = %[9]q
  users     = %[10]s
  groups    = %[11]s

  depends_on = [
    pulp_user.alice,
    pulp_user.bob,
    pulp_user.carol,
    pulp_group.buddies,
    pulp_group.friends,
  ]
}
`,
		cgName, distName, basePath,
		userAlice, userBob, userCarol,
		groupBuddies, groupFriends,
		role,
		hclStringList(users),
		hclStringList(groups),
	)
}

// hclStringList renders a Go []string as an HCL list literal, e.g. `["a", "b"]`.
func hclStringList(ss []string) string {
	if len(ss) == 0 {
		return "[]"
	}
	quoted := make([]string, len(ss))
	for i, s := range ss {
		quoted[i] = fmt.Sprintf("%q", s)
	}
	return "[" + join(quoted, ", ") + "]"
}

func join(ss []string, sep string) string {
	out := ""
	for i, s := range ss {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
}
