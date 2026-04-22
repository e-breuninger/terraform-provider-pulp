// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestUserResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
			resource "pulp_user" "breuninger" {
				username = "breuninger"
				password = "supersecret"
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pulp_user.breuninger", "username", "breuninger"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "pulp_user.breuninger",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIgnore:              []string{"password"},
				ImportStateVerifyIdentifierAttribute: "pulp_href",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources["pulp_user.breuninger"]
					if !ok {
						return "", fmt.Errorf("resource not found: pulp_user.breuninger")
					}
					return rs.Primary.Attributes["pulp_href"], nil
				},
			},
			// Update and Read testing
			{
				Config: providerConfig + `
			resource "pulp_user" "breuninger" {
				username   = "e.breuninger"
				password   = "supersecret123"
				first_name = "E."
				last_name  = "Breuninger"
				email      = "info@breuninger.de"
				is_staff   = true
				is_active  = true
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pulp_user.breuninger", "username", "e.breuninger"),
					resource.TestCheckResourceAttr("pulp_user.breuninger", "first_name", "E."),
					resource.TestCheckResourceAttr("pulp_user.breuninger", "last_name", "Breuninger"),
					resource.TestCheckResourceAttr("pulp_user.breuninger", "email", "info@breuninger.de"),
					resource.TestCheckResourceAttr("pulp_user.breuninger", "is_staff", "true"),
					resource.TestCheckResourceAttr("pulp_user.breuninger", "is_active", "true"),
					resource.TestCheckResourceAttrSet("pulp_user.breuninger", "password"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
