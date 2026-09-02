// Copyright E. Breuninger GmbH & Co 2026
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
	ServerUrl types.String `tfsdk:"server_url"`
	Username  types.String `tfsdk:"username"`
	Password  types.String `tfsdk:"password"`
	ForceIPv4 types.Bool   `tfsdk:"force_ipv4"`
}

func (p *PulpProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "pulp"
	resp.Version = p.version
}

func (p *PulpProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			// Optional, not Required: a Required attribute cannot be omitted
			// in favour of the environment variable.
			"server_url": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "URI for Pulp API. May also be provided via the `PULP_SERVER_URL` environment variable.",
			},
			"username": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Username for Pulp API. May also be provided via the `PULP_USERNAME` environment variable.",
			},
			"password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Password for Pulp API. May also be provided via the `PULP_PASSWORD` environment variable.",
			},
			"force_ipv4": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether to force the provider to use IPv4 when connecting to the Pulp API. May also be provided via `PULP_FORCE_IPV4` environment variable.",
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

	settings := []struct {
		attribute  string
		env        string
		configured types.String
		value      string
	}{
		{attribute: "server_url", env: "PULP_SERVER_URL", configured: config.ServerUrl},
		{attribute: "username", env: "PULP_USERNAME", configured: config.Username},
		{attribute: "password", env: "PULP_PASSWORD", configured: config.Password},
	}
	for i, s := range settings {
		if s.configured.IsUnknown() {
			resp.Diagnostics.AddAttributeError(path.Root(s.attribute),
				fmt.Sprintf("Unknown Pulp API %s", s.attribute),
				fmt.Sprintf("The provider cannot be configured from a value that is only known after apply. "+
					"Set %s to a known value or use the %s environment variable.", s.attribute, s.env))
			continue
		}
		if !s.configured.IsNull() {
			settings[i].value = s.configured.ValueString()
			continue
		}
		if settings[i].value = os.Getenv(s.env); settings[i].value == "" {
			resp.Diagnostics.AddAttributeError(path.Root(s.attribute),
				fmt.Sprintf("Missing Pulp API %s", s.attribute),
				fmt.Sprintf("Set the provider's %s attribute or the %s environment variable.", s.attribute, s.env))
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}
	serverURL, username, password := settings[0].value, settings[1].value, settings[2].value

	forceIPv4 := os.Getenv("PULP_FORCE_IPV4") == "true"
	if !config.ForceIPv4.IsNull() && !config.ForceIPv4.IsUnknown() {
		forceIPv4 = config.ForceIPv4.ValueBool()
	}

	parsedURL, err := urlx.Parse(serverURL)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("server_url"), "No valid URL.",
			fmt.Sprintf("Error while trying to parse URL: %s", err))
		return
	}

	pulpClient, err := client.NewPulpClient(parsedURL.String(), username, password, forceIPv4)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Pulp client", fmt.Sprintf("Error: %s", err))
		return
	}

	resp.DataSourceData = pulpClient
	resp.ResourceData = pulpClient
}

func (p *PulpProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewPulpRemoteResource,
		NewPulpRepositoryResource,
		NewPulpDistributionResource,
		NewPulpContentGuardResource,
		NewPulpGroupResource,
		NewPulpUserResource,
		NewPulpRoleResource,
		NewPulpUserRoleResource,
		NewPulpObjectRoleResource,
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
