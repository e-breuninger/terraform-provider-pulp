// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
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

func TestRemoteDownloadSettings(t *testing.T) {
	name := fmt.Sprintf("tf-acc-remote-download-%s", internal.RandomSuffix())

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{ // set
				Config: providerConfig + fmt.Sprintf(`
resource "pulp_remote" "download" {
  content_type         = "npm"
  plugin_name          = "npm"
  url                  = "https://registry.npmjs.org/"
  name                 = %[1]q
  download_concurrency = 5
  max_retries          = 0
  rate_limit           = 10
  total_timeout        = 300
  connect_timeout      = 0.1
  sock_connect_timeout = 2.5
  sock_read_timeout    = 60
}`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pulp_remote.download", "download_concurrency", "5"),
					resource.TestCheckResourceAttr("pulp_remote.download", "max_retries", "0"),
					resource.TestCheckResourceAttr("pulp_remote.download", "rate_limit", "10"),
					resource.TestCheckResourceAttr("pulp_remote.download", "total_timeout", "300"),
					resource.TestCheckResourceAttr("pulp_remote.download", "connect_timeout", "0.1"),
					resource.TestCheckResourceAttr("pulp_remote.download", "sock_connect_timeout", "2.5"),
					resource.TestCheckResourceAttr("pulp_remote.download", "sock_read_timeout", "60"),
				),
			},
			{ // clear -> Pulp must receive an explicit null
				Config: providerConfig + fmt.Sprintf(`
resource "pulp_remote" "download" {
  content_type = "npm"
  plugin_name  = "npm"
  url          = "https://registry.npmjs.org/"
  name         = %[1]q
}`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr("pulp_remote.download", "download_concurrency"),
					resource.TestCheckNoResourceAttr("pulp_remote.download", "max_retries"),
					resource.TestCheckNoResourceAttr("pulp_remote.download", "rate_limit"),
					resource.TestCheckNoResourceAttr("pulp_remote.download", "total_timeout"),
					resource.TestCheckNoResourceAttr("pulp_remote.download", "connect_timeout"),
					resource.TestCheckNoResourceAttr("pulp_remote.download", "sock_connect_timeout"),
					resource.TestCheckNoResourceAttr("pulp_remote.download", "sock_read_timeout"),
				),
			},
		},
	})
}

func TestRemoteFileGit(t *testing.T) {
	name := fmt.Sprintf("tf-acc-remote-git-%s", internal.RandomSuffix())
	config := func(gitRef string) string {
		return providerConfig + fmt.Sprintf(`
resource "pulp_remote" "git" {
  content_type = "file"
  plugin_name  = "git"
  url          = "https://github.com/pulp/pulp-glue.git"
  name         = %[1]q
  %[2]s
}`, name, gitRef)
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{ // Pulp defaults the ref
				Config: config(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pulp_remote.git", "git_ref", "HEAD"),
					resource.TestCheckNoResourceAttr("pulp_remote.git", "policy"),
				),
			},
			{
				Config: config(`git_ref = "main"`),
				Check:  resource.TestCheckResourceAttr("pulp_remote.git", "git_ref", "main"),
			},
			{
				ResourceName:                         "pulp_remote.git",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "pulp_href",
				ImportStateIdFunc: func(state *terraform.State) (string, error) {
					return state.RootModule().Resources["pulp_remote.git"].Primary.Attributes["pulp_href"], nil
				},
			},
		},
	})
}

func TestRemoteGitRefNeedsGitVariant(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "pulp_remote" "file" {
  content_type = "file"
  plugin_name  = "file"
  url          = "https://example.com/PULP_MANIFEST"
  name         = "tf-acc-remote-git-ref"
  git_ref      = "main"
}`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Attribute "git_ref" is not supported by this variant`),
			},
		},
	})
}
