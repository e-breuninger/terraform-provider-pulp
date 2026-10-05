// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestUserRoleResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
			resource "pulp_role" "filerepository" {
				name        = "file.filerepository"
				description = "Allow viewing the file repository"
				permissions = [
					"file.view_filerepository"
				]
			}

			resource "pulp_user" "breuninger" {
				username = "breuninger"
				password = "supersecret"
			}

			resource "pulp_user_role" "breuninger_filerepository" {
				role = pulp_role.filerepository.name
				user_id = pulp_user.breuninger.id
				content_object = null
				content_object_prn = null
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pulp_role.filerepository", "permissions.#", "1"),
					resource.TestCheckResourceAttr("pulp_user_role.breuninger_filerepository", "role", "file.filerepository"),
					resource.TestCheckResourceAttrSet("pulp_user_role.breuninger_filerepository", "pulp_href"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "pulp_user_role.breuninger_filerepository",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "pulp_href",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources["pulp_user_role.breuninger_filerepository"]
					if !ok {
						return "", fmt.Errorf("resource not found: pulp_user_role.breuninger_filerepository")
					}
					return rs.Primary.Attributes["pulp_href"], nil
				},
			},
			// Update and Read testing
			{
				Config: providerConfig + `
			resource "pulp_role" "filerepository" {
				name        = "file.filerepository"
				description = "Allow viewing the file repository"
				permissions = [
					"file.view_filerepository",
					"file.delete_filerepository",
				]
			}

			resource "pulp_user" "breuninger" {
				username = "breuninger"
				password = "supersecret"
			}

			resource "pulp_user_role" "breuninger_filerepository" {
				role 						= pulp_role.filerepository.name
				user_id 				= pulp_user.breuninger.id
				content_object 	= null
				content_object_prn = null
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pulp_role.filerepository", "permissions.#", "2"),
					resource.TestCheckResourceAttrSet("pulp_user_role.breuninger_filerepository", "pulp_href"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

// TestUserRoleContentObjectResource sets one of content_object and
// content_object_prn and expects Pulp to fill in the other.
func TestUserRoleContentObjectResource(t *testing.T) {
	const config = `
			resource "pulp_repository" "owned" {
				content_type = "file"
				description  = "Owned file repository"
				plugin_name  = "file"
				name         = "user-role-owned"
			}

			resource "pulp_user" "owner" {
				username = "user-role-owner"
				password = "supersecret"
			}
			`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + config + `
			resource "pulp_user_role" "by_href" {
				role           = "file.filerepository_owner"
				user_id        = pulp_user.owner.id
				content_object = pulp_repository.owned.pulp_href
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("pulp_user_role.by_href", "content_object",
						"pulp_repository.owned", "pulp_href"),
					resource.TestMatchResourceAttr("pulp_user_role.by_href", "content_object_prn",
						regexp.MustCompile(`^prn:file\.filerepository:`)),
				),
			},
			{
				Config: providerConfig + config + `
			resource "pulp_user_role" "by_prn" {
				role               = "file.filerepository_viewer"
				user_id            = pulp_user.owner.id
				content_object_prn = format("prn:file.filerepository:%s",
					element(split("/", trimsuffix(pulp_repository.owned.pulp_href, "/")), 7))
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("pulp_user_role.by_prn", "content_object",
						"pulp_repository.owned", "pulp_href"),
				),
			},
		},
	})
}
