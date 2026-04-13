// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestDistributionResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + `
			resource "pulp_remote" "npm_dist_test" {
				content_type = "npm"
				plugin_name  = "npm"
				url          = "https://registry.npmjs.org/"
				name         = "npm_dist_test"
			}

			resource "pulp_distribution" "npm" {
				content_type = "npm"
				plugin_name  = "npm"
				name         = "baz"
				base_path    = "foo"
				remote       = pulp_remote.npm_dist_test.pulp_href
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pulp_distribution.npm", "name", "baz"),
					resource.TestCheckResourceAttr("pulp_distribution.npm", "base_path", "foo"),
					resource.TestCheckResourceAttrSet("pulp_distribution.npm", "pulp_href"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "pulp_distribution.npm",
				ImportState:       true,
				ImportStateVerify: true,
				// The last_updated attribute does not exist in the HashiCups
				// API, therefore there is no value for it during import.
				ImportStateVerifyIgnore:              []string{"last_updated"},
				ImportStateVerifyIdentifierAttribute: "pulp_href",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources["pulp_distribution.npm"]
					if !ok {
						return "", fmt.Errorf("resource not found: pulp_distribution.npm")
					}
					return rs.Primary.Attributes["pulp_href"], nil
				},
			},
			// Update and Read testing
			{
				Config: providerConfig + `
			resource "pulp_remote" "npm_dist_test" {
				content_type = "npm"
				plugin_name  = "npm"
				url          = "https://registry.npmjs.org/"
				name         = "npm_dist_test"
			}

			resource "pulp_distribution" "npm" {
				content_type = "npm"
				plugin_name  = "npm"
				name         = "foo"
				base_path    = "foo"
				remote       = pulp_remote.npm_dist_test.pulp_href
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pulp_distribution.npm", "name", "foo"),
					resource.TestCheckResourceAttr("pulp_distribution.npm", "base_path", "foo"),
					resource.TestCheckResourceAttrSet("pulp_distribution.npm", "pulp_href"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
