// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestDistributionDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "pulp_remote" "npm_dist_source_test" {
	content_type = "npm"
	plugin_name  = "npm"
	url          = "https://registry.npmjs.org/"
	name         = "npm_dist_source_test"
}

resource "pulp_distribution" "npm_source_test" {
	content_type = "npm"
	plugin_name  = "npm"
	name         = "npm-dist-source-test"
	base_path    = "npm-dist-source-test"
	remote       = pulp_remote.npm_dist_source_test.pulp_href
}

data "pulp_distribution" "npm_source_test" {
	content_type = "npm"
	plugin_name  = "npm"
	name         = pulp_distribution.npm_source_test.name
	base_path    = pulp_distribution.npm_source_test.base_path
}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.pulp_distribution.npm_source_test", "name", "npm-dist-source-test"),
					resource.TestCheckResourceAttr("data.pulp_distribution.npm_source_test", "base_path", "npm-dist-source-test"),
					resource.TestCheckResourceAttrSet("data.pulp_distribution.npm_source_test", "pulp_href"),
				),
			},
		},
	})
}
