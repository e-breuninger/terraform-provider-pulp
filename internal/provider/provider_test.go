// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const (
	// defaultTestServerURL is the Pulp the acceptance tests and the API
	// schema conformance tests run against. Override with PULP_SERVER_URL.
	defaultTestServerURL = "http://localhost:8080"

	providerConfig = `
provider "pulp" {
  username   = "admin"
  password   = "admin"
  server_url = "` + defaultTestServerURL + `"
}
`
)

var (
	// testAccProtoV6ProviderFactories are used to instantiate a provider during
	// acceptance testing. The factory function will be invoked for every Terraform
	// CLI command executed to create a provider server to which the CLI can
	// reattach.
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"pulp": providerserver.NewProtocol6WithError(New("test")()),
	}
)

// TestProviderFromEnvironment checks that the documented environment
// variables can stand in for the provider attributes. They could not while
// those attributes were Required: Terraform rejected the config before
// Configure ever ran.
func TestProviderFromEnvironment(t *testing.T) {
	t.Setenv("PULP_SERVER_URL", defaultTestServerURL)
	t.Setenv("PULP_USERNAME", "admin")
	t.Setenv("PULP_PASSWORD", "admin")

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
provider "pulp" {}

resource "pulp_group" "env" {
  name = "tf-acc-provider-env"
}`,
			Check: resource.TestCheckResourceAttrSet("pulp_group.env", "pulp_href"),
		}},
	})
}

// TestProviderMissingCredentials checks that an unconfigured provider names
// both ways of supplying each setting.
func TestProviderMissingCredentials(t *testing.T) {
	for _, env := range []string{"PULP_SERVER_URL", "PULP_USERNAME", "PULP_PASSWORD"} {
		t.Setenv(env, "")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      `provider "pulp" {}` + "\n" + `resource "pulp_group" "none" { name = "unused" }`,
			ExpectError: regexp.MustCompile(`Set the provider's server_url attribute or the PULP_SERVER_URL`),
		}},
	})
}
