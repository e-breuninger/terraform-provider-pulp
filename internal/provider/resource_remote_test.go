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

func TestRemoteResource(t *testing.T) {
	suffix := internal.RandomSuffix()
	remoteName1 := fmt.Sprintf("tf-acc-remote-%s", suffix)
	remoteName2 := fmt.Sprintf("tf-acc-remote-%s", suffix)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
resource "pulp_remote" "npm" {
  content_type = "npm"
  plugin_name  = "npm"
  url          = "https://registry.npmjs.org/"
  name         = %[1]q
}`, remoteName1),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pulp_remote.npm", "url", "https://registry.npmjs.org/"),
					resource.TestCheckResourceAttr("pulp_remote.npm", "name", remoteName1),
					resource.TestCheckResourceAttrSet("pulp_remote.npm", "pulp_href"),
				),
			},
			// ImportState testing
			{
				ResourceName:                         "pulp_remote.npm",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIgnore:              []string{"last_updated"},
				ImportStateVerifyIdentifierAttribute: "pulp_href",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					rs, ok := state.RootModule().Resources["pulp_remote.npm"]
					if !ok {
						return "", fmt.Errorf("resource not found: pulp_remote.npm")
					}
					return rs.Primary.Attributes["pulp_href"], nil
				},
			},
			// Update and Read testing
			{
				Config: providerConfig + fmt.Sprintf(`
resource "pulp_remote" "npm" {
  content_type = "npm"
  plugin_name  = "npm"
  url          = "https://registry.npmjs.org"
  name         = %[1]q
}`, remoteName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pulp_remote.npm", "url", "https://registry.npmjs.org"),
					resource.TestCheckResourceAttr("pulp_remote.npm", "name", remoteName2),
					resource.TestCheckResourceAttrSet("pulp_remote.npm", "pulp_href"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}
