// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestRemotesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "pulp_remote" "npm_remotes_source_test" {
	content_type = "npm"
	plugin_name  = "npm"
	url          = "https://registry.npmjs.org/"
	name         = "npm_remotes_source_test"
}

data "pulp_remotes" "npm_remotes_source_test" {
	content_type = "npm"
	plugin_name  = "npm"
	name         = pulp_remote.npm_remotes_source_test.name

	depends_on = [pulp_remote.npm_remotes_source_test]
}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.pulp_remotes.npm_remotes_source_test", "remotes.#", "1"),
					resource.TestCheckResourceAttr("data.pulp_remotes.npm_remotes_source_test", "remotes.0.name", "npm_remotes_source_test"),
					resource.TestCheckResourceAttr("data.pulp_remotes.npm_remotes_source_test", "remotes.0.url", "https://registry.npmjs.org/"),
					resource.TestCheckResourceAttrSet("data.pulp_remotes.npm_remotes_source_test", "remotes.0.pulp_href"),
				),
			},
		},
	})
}
