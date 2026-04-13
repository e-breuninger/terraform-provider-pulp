// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestRepositoryResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
			resource "pulp_repository" "npm" {
			  content_type = "npm"
			  plugin_name  = "npm"
				name         = "foo"
				description  = "npm repository"
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pulp_repository.npm", "name", "foo"),
					resource.TestCheckResourceAttr("pulp_repository.npm", "description", "npm repository"),
					resource.TestCheckResourceAttrSet("pulp_repository.npm", "pulp_href"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "pulp_repository.npm",
				ImportState:       true,
				ImportStateVerify: true,
				// The last_updated attribute does not exist in the HashiCups
				// API, therefore there is no value for it during import.
				ImportStateVerifyIgnore:              []string{"last_updated"},
				ImportStateVerifyIdentifierAttribute: "pulp_href",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources["pulp_repository.npm"]
					if !ok {
						return "", fmt.Errorf("resource not found: pulp_repository.npm")
					}
					return rs.Primary.Attributes["pulp_href"], nil
				},
			},
			// Update and Read testing
			{
				Config: providerConfig + `
			resource "pulp_repository" "npm" {
			  content_type = "npm"
			  plugin_name  = "npm"
				name         = "foo"
				description  = "updated npm repository"
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pulp_repository.npm", "name", "foo"),
					resource.TestCheckResourceAttr("pulp_repository.npm", "description", "updated npm repository"),
					resource.TestCheckResourceAttrSet("pulp_repository.npm", "pulp_href"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
