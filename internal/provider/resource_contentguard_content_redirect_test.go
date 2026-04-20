// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestContentGuardResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create and Read
			{
				Config: providerConfig + `
resource "pulp_contentguard" "test" {
  content_type = "core"
  plugin_name  = "content_redirect"
  name         = "test-guard"
  description  = "initial description"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pulp_contentguard.test", "name", "test-guard"),
					resource.TestCheckResourceAttr("pulp_contentguard.test", "description", "initial description"),
					resource.TestCheckResourceAttr("pulp_contentguard.test", "content_type", "core"),
					resource.TestCheckResourceAttr("pulp_contentguard.test", "plugin_name", "content_redirect"),
					resource.TestCheckResourceAttrSet("pulp_contentguard.test", "pulp_href"),
				),
			},
			// Step 2: Update (same resource, changed description)
			{
				Config: providerConfig + `
resource "pulp_contentguard" "test" {
  content_type = "core"
  plugin_name  = "content_redirect"
  name         = "test-guard"
  description  = "updated description"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pulp_contentguard.test", "name", "test-guard"),
					resource.TestCheckResourceAttr("pulp_contentguard.test", "description", "updated description"),
					resource.TestCheckResourceAttrSet("pulp_contentguard.test", "pulp_href"),
				),
			},
			// Step 3: ImportState
			{
				ResourceName:            "pulp_contentguard.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{},
			},
			// Step 4: Delete automatically occurs at end of TestCase
		},
	})
}

func TestContentGuardResource_ImportFromPullThrough(t *testing.T) {
	var contentGuardHref string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create the pull-through remote + distribution,
			// which implicitly creates a content_redirect content guard
			{
				Config: providerConfig + `
resource "pulp_remote" "docker" {
  content_type = "container"
  plugin_name  = "pull-through"
  url          = "https://registry-1.docker.io"
  name         = "docker"
}

resource "pulp_distribution" "docker" {
  content_type = "container"
  plugin_name  = "pull-through"
  name         = "docker"
  base_path    = "docker"
  remote       = pulp_remote.docker.pulp_href
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("pulp_distribution.docker", "content_guard"),
					// Capture the content_guard href for the import step
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["pulp_distribution.docker"]
						if !ok {
							return fmt.Errorf("resource not found: pulp_distribution.docker")
						}
						contentGuardHref = rs.Primary.Attributes["content_guard"]
						if contentGuardHref == "" {
							return fmt.Errorf("content_guard attribute is empty")
						}
						return nil
					},
				),
			},
			// Step 2: Add the contentguard resource to config + import it
			{
				Config: providerConfig + `
resource "pulp_remote" "docker" {
  content_type = "container"
  plugin_name  = "pull-through"
  url          = "https://registry-1.docker.io"
  name         = "docker"
}

resource "pulp_distribution" "docker" {
  content_type = "container"
  plugin_name  = "pull-through"
  name         = "docker"
  base_path    = "docker"
  remote       = pulp_remote.docker.pulp_href
}

resource "pulp_contentguard" "docker" {
  content_type = "core"
  plugin_name  = "content_redirect"
  name         = "docker"
	description  = "content guard for docker"
}
`,
				ResourceName:                         "pulp_contentguard.docker",
				ImportState:                          true,
				ImportStateVerifyIdentifierAttribute: "pulp_href",
				ImportStateIdFunc: func(_ *terraform.State) (string, error) {
					if contentGuardHref == "" {
						return "", fmt.Errorf("content_guard href was not captured from step 1")
					}
					return contentGuardHref, nil
				},
			},
			// Step 3: Verify the imported resource is now fully managed
			{
				Config: providerConfig + `
resource "pulp_remote" "docker" {
  content_type = "container"
  plugin_name  = "pull-through"
  url          = "https://registry-1.docker.io"
  name         = "docker"
}

resource "pulp_distribution" "docker" {
  content_type = "container"
  plugin_name  = "pull-through"
  name         = "docker"
  base_path    = "docker"
  remote       = pulp_remote.docker.pulp_href
}

resource "pulp_contentguard" "docker" {
  content_type = "core"
  plugin_name  = "content_redirect"
  name         = "docker"
	description  = "content guard for docker"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("pulp_contentguard.docker", "pulp_href"),
					resource.TestCheckResourceAttr("pulp_contentguard.docker", "content_type", "core"),
					resource.TestCheckResourceAttr("pulp_contentguard.docker", "plugin_name", "content_redirect"),
				),
			},
		},
	})
}
