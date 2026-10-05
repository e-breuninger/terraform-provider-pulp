// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"github.com/e-breuninger/terraform-provider-pulp/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type PulpRemoteModel struct {
	PulpHref      types.String `tfsdk:"pulp_href"`
	Prn           types.String `tfsdk:"prn"`
	ContentType   types.String `tfsdk:"content_type"`
	PluginName    types.String `tfsdk:"plugin_name"`
	Name          types.String `tfsdk:"name"`
	Url           types.String `tfsdk:"url"`
	Policy        types.String `tfsdk:"policy"`
	TlsValidation types.Bool   `tfsdk:"tls_validation"`
	Username      types.String `tfsdk:"username"`
	Password      types.String `tfsdk:"password"`
	PulpLabels    types.Map    `tfsdk:"pulp_labels"`

	DownloadConcurrency types.Number `tfsdk:"download_concurrency"`
	MaxRetries          types.Number `tfsdk:"max_retries"`
	RateLimit           types.Number `tfsdk:"rate_limit"`
	TotalTimeout        types.Number `tfsdk:"total_timeout"`
	ConnectTimeout      types.Number `tfsdk:"connect_timeout"`
	SockConnectTimeout  types.Number `tfsdk:"sock_connect_timeout"`
	SockReadTimeout     types.Number `tfsdk:"sock_read_timeout"`
}

type pulpRemoteResource struct {
	pulpResource[PulpRemoteModel]
}

func NewPulpRemoteResource() resource.Resource {
	return &pulpRemoteResource{pulpResource[PulpRemoteModel]{
		typeName:    "remote",
		label:       "Remote",
		description: "Manages a Pulp Remote for any content type.",
		collection:  "remotes",
		features:    remoteFeatures,
		fields: variantResourceFields(remoteFeatures,
			field{
				Name: "name", Kind: fieldString, Required: true,
				Description: "A unique name for this Remote.",
			},
			field{
				Name: "url", Kind: fieldString, Required: true,
				Description: "The URL of an external content source.",
			},
			field{
				Name: "policy", Kind: fieldString,
				Optional: true, Computed: true, Feature: featurePolicy,
				Description: "Download policy: `immediate`, `on_demand`, or `streamed`.",
				StringValidators: []validator.String{
					stringvalidator.OneOf("immediate", "on_demand", "streamed"),
				},
			},
			field{
				Name: "tls_validation", Kind: fieldBool,
				Optional: true, Computed: true,
				Description: "Whether TLS peer validation must be performed.",
			},
			// Write-only: Pulp never reports credentials back.
			field{
				Name: "username", Kind: fieldString,
				Optional: true, WriteOnly: true,
				Description: "Username for authentication when syncing.",
			},
			field{
				Name: "password", Kind: fieldString,
				Optional: true, Sensitive: true, WriteOnly: true,
				Description: "Password for authentication when syncing.",
			},
			labelsField(),
			downloadField("download_concurrency", 1,
				"Total number of simultaneous connections. Unset uses Pulp's default."),
			downloadField("max_retries", 0,
				"Maximum number of retry attempts after a download failure. Unset uses Pulp's default of 3."),
			downloadField("rate_limit", 0,
				"Limits requests per second for each concurrent downloader."),
			downloadField("total_timeout", 0,
				"Total timeout for a download in seconds. Unset uses aiohttp's default."),
			downloadField("connect_timeout", 0,
				"Timeout in seconds for acquiring a connection from the pool. Unset uses aiohttp's default."),
			downloadField("sock_connect_timeout", 0,
				"Timeout in seconds for connecting to a peer for a new connection. Unset uses aiohttp's default."),
			downloadField("sock_read_timeout", 0,
				"Timeout in seconds for reading a portion of data from a peer. Unset uses aiohttp's default."),
		),
	}}
}

// downloadField declares a download tuning setting. Pulp accepts an explicit
// null, so removing it from the config restores the default.
func downloadField(name string, minimum int64, description string) field {
	return field{
		Name: name, Kind: fieldNumber,
		Optional: true, Nullable: true,
		Description:      description,
		NumberValidators: []validator.Number{validators.NumberAtLeast(minimum)},
	}
}
