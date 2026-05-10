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
				ResourceName:                         "pulp_repository.npm",
				ImportState:                          true,
				ImportStateVerify:                    true,
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

func TestRepositoryResourceRetainVersions(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Validator: value 0 must be rejected at plan time (no API call made)
			{
				Config: providerConfig + `
			resource "pulp_repository" "npm_retain" {
			  content_type         = "npm"
			  plugin_name          = "npm"
			  name                 = "foo-retain"
			  description          = "npm repository with version cap"
			  retain_repo_versions = 0
			}
			`,
				ExpectError: regexp.MustCompile(`Attribute retain_repo_versions value must be at least 1`),
			},
			// Step A: Create with retain_repo_versions = 5
			{
				Config: providerConfig + `
			resource "pulp_repository" "npm_retain" {
			  content_type         = "npm"
			  plugin_name          = "npm"
			  name                 = "foo-retain"
			  description          = "npm repository with version cap"
			  retain_repo_versions = 5
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pulp_repository.npm_retain", "name", "foo-retain"),
					resource.TestCheckResourceAttr("pulp_repository.npm_retain", "description", "npm repository with version cap"),
					resource.TestCheckResourceAttr("pulp_repository.npm_retain", "retain_repo_versions", "5"),
					resource.TestCheckResourceAttrSet("pulp_repository.npm_retain", "pulp_href"),
				),
			},
			// Step B: Update retain_repo_versions to 10
			{
				Config: providerConfig + `
			resource "pulp_repository" "npm_retain" {
			  content_type         = "npm"
			  plugin_name          = "npm"
			  name                 = "foo-retain"
			  description          = "npm repository with version cap"
			  retain_repo_versions = 10
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pulp_repository.npm_retain", "retain_repo_versions", "10"),
				),
			},
			// Step C: ImportState — retain_repo_versions must round-trip through Read
			{
				ResourceName:                         "pulp_repository.npm_retain",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIgnore:              []string{"last_updated"},
				ImportStateVerifyIdentifierAttribute: "pulp_href",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources["pulp_repository.npm_retain"]
					if !ok {
						return "", fmt.Errorf("resource not found: pulp_repository.npm_retain")
					}
					return rs.Primary.Attributes["pulp_href"], nil
				},
			},
			// Step D: Remove retain_repo_versions (clear the cap — Pulp returns null).
			// Without Computed, a null Optional Int64 is absent from state entirely.
			{
				Config: providerConfig + `
			resource "pulp_repository" "npm_retain" {
			  content_type = "npm"
			  plugin_name  = "npm"
			  name         = "foo-retain"
			  description  = "npm repository with version cap"
			}
			`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("pulp_repository.npm_retain", "retain_repo_versions"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
