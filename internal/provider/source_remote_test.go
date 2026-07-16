// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestRemoteDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + `
resource "pulp_remote" "npm_source_test" {
	content_type = "npm"
	plugin_name  = "npm"
	url          = "https://registry.npmjs.org/"
	name         = "npm_remote_source_test"
}

data "pulp_remote" "npm_source_test" {
	pulp_href = pulp_remote.npm_source_test.pulp_href
}
				`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.pulp_remote.npm_source_test", "name", "npm_remote_source_test"),
					resource.TestCheckResourceAttr("data.pulp_remote.npm_source_test", "url", "https://registry.npmjs.org/"),
					resource.TestCheckResourceAttrSet("data.pulp_remote.npm_source_test", "pulp_href"),
				),
			},
		},
	})
}
