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
			// Step 4: Delete automatically occurs at end of TestCase
		},
	})
}

func TestContentGuardResource_ImportFromPullThrough(t *testing.T) {
	var contentGuardHref string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
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
  name         = "content redirect"
}
`,
				ResourceName: "pulp_contentguard.docker",
				ImportState:  true,
				ImportStateIdFunc: func(_ *terraform.State) (string, error) {
					return contentGuardHref, nil
				},
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected 1 state, got %d", len(states))
					}
					attrs := states[0].Attributes
					if attrs["pulp_href"] == "" {
						return fmt.Errorf("pulp_href is empty")
					}
					if attrs["name"] != "content redirect" {
						return fmt.Errorf("expected name 'content redirect', got '%s'", attrs["name"])
					}
					return nil
				},
			},
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
					resource.TestCheckResourceAttr("pulp_contentguard.docker", "name", "docker"),
					resource.TestCheckResourceAttr("pulp_contentguard.docker", "description", "content guard for docker"),
				),
			},
			// Step 4: Automatic destroy
		},
	})
}
