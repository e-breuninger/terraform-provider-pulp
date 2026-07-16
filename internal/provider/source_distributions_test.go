// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestDistributionsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "pulp_remote" "npm_dists_source_test" {
	content_type = "npm"
	plugin_name  = "npm"
	url          = "https://registry.npmjs.org/"
	name         = "npm_dists_source_test"
}

resource "pulp_distribution" "npm_dists_source_test" {
	content_type = "npm"
	plugin_name  = "npm"
	name         = "npm-dists-source-test"
	base_path    = "npm-dists-source-test"
	remote       = pulp_remote.npm_dists_source_test.pulp_href
}

data "pulp_distributions" "npm_dists_source_test" {
	content_type = "npm"
	plugin_name  = "npm"
	base_path    = pulp_distribution.npm_dists_source_test.base_path

	depends_on = [pulp_distribution.npm_dists_source_test]
}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.pulp_distributions.npm_dists_source_test", "distributions.#", "1"),
					resource.TestCheckResourceAttr("data.pulp_distributions.npm_dists_source_test", "distributions.0.name", "npm-dists-source-test"),
					resource.TestCheckResourceAttr("data.pulp_distributions.npm_dists_source_test", "distributions.0.base_path", "npm-dists-source-test"),
					resource.TestCheckResourceAttrSet("data.pulp_distributions.npm_dists_source_test", "distributions.0.pulp_href"),
				),
			},
		},
	})
}
