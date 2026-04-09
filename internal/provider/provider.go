// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/e-breuninger/terraform-provider-pulp/internal/client"
	"github.com/goware/urlx"
	"github.com/hashicorp/terraform-plugin-framework/action"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure PulpProvider satisfies various provider interfaces.
var _ provider.Provider = &PulpProvider{}
var _ provider.ProviderWithFunctions = &PulpProvider{}
var _ provider.ProviderWithEphemeralResources = &PulpProvider{}
var _ provider.ProviderWithActions = &PulpProvider{}

// PulpProvider defines the provider implementation.
type PulpProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// PulpProviderModel describes the provider data model.
type PulpProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
}

func (p *PulpProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "pulp"
	resp.Version = p.version
}

func (p *PulpProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "URI for Pulp API. May also be provided via PULP_ENDPOINT environment variable.",
			},
			"username": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Username for Pulp API. May also be provided via PULP_USERNAME environment variable.",
			},
			"password": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				MarkdownDescription: "Password for Pulp API. May also be provided via PULP_PASSWORD environment variable.",
			},
		},
	}
}

func (p *PulpProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config PulpProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := os.Getenv("PULP_ENDPOINT")
	username := os.Getenv("PULP_USERNAME")
	password := os.Getenv("PULP_PASSWORD")

	if !config.Endpoint.IsNull() {
		endpoint = config.Endpoint.ValueString()
	}

	parsedURL, urlParseError := urlx.Parse(endpoint)
	if urlParseError != nil {
		resp.Diagnostics.AddAttributeError(path.Root("endpoint"), "No valid URL.", fmt.Sprintf("Error while trying to parse URL: %s", urlParseError))
		return
	}

	if !config.Username.IsNull() {
		username = config.Username.ValueString()
	}

	if !config.Password.IsNull() {
		password = config.Password.ValueString()
	}

	pulpClient := client.NewPulpClient(
		parsedURL.String(),
		username, password,
	)

	resp.DataSourceData = pulpClient
	resp.ResourceData = pulpClient
}

func (p *PulpProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewPulpRemoteResource,
		NewPulpRepositoryResource,
		NewPulpDistributionResource,
	}
}

func (p *PulpProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
}

func (p *PulpProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *PulpProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func (p *PulpProvider) Actions(ctx context.Context) []func() action.Action {
	return []func() action.Action{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &PulpProvider{
			version: version,
		}
	}
}
