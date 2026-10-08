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
	GitRef        types.String `tfsdk:"git_ref"`
	TlsValidation types.Bool   `tfsdk:"tls_validation"`
	CaCert        types.String `tfsdk:"ca_cert"`
	ClientCert    types.String `tfsdk:"client_cert"`
	ClientKey     types.String `tfsdk:"client_key"`
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
				Name: "git_ref", Kind: fieldString,
				Optional: true, Computed: true, Feature: featureGitRef,
				Description: "The git ref (branch, tag, or commit hash) to sync from. Unset uses Pulp's default.",
			},
			field{
				Name: "tls_validation", Kind: fieldBool,
				Optional: true, Computed: true,
				Description: "Whether TLS peer validation must be performed.",
			},
			field{
				Name: "ca_cert", Kind: fieldString,
				Optional: true, Nullable: true, Certificate: true,
				Description:      "A PEM encoded CA certificate used to validate the server certificate presented by the remote server.",
				StringValidators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			field{
				Name: "client_cert", Kind: fieldString,
				Optional: true, Nullable: true, Certificate: true,
				Description:      "A PEM encoded client certificate used for authentication.",
				StringValidators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			// Write-only: Pulp never reports credentials back.
			field{
				Name: "client_key", Kind: fieldString,
				Optional: true, Nullable: true, Sensitive: true, WriteOnly: true,
				Description:      "A PEM encoded private key used for authentication.",
				StringValidators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
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
			field{
				Name: "download_concurrency", Kind: fieldNumber,
				Optional: true, Nullable: true,
				Description:      "Total number of simultaneous connections. Unset uses Pulp's default.",
				NumberValidators: []validator.Number{validators.NumberAtLeast(1)},
			},
			field{
				Name: "max_retries", Kind: fieldNumber,
				Optional: true, Nullable: true,
				Description:      "Maximum number of retry attempts after a download failure. Unset uses Pulp's default of 3.",
				NumberValidators: []validator.Number{validators.NumberAtLeast(0)},
			},
			field{
				Name: "rate_limit", Kind: fieldNumber,
				Optional: true, Nullable: true,
				Description:      "Limits requests per second for each concurrent downloader.",
				NumberValidators: []validator.Number{validators.NumberAtLeast(0)},
			},
			field{
				Name: "total_timeout", Kind: fieldNumber,
				Optional: true, Nullable: true,
				Description:      "Total timeout for a download in seconds. Unset uses aiohttp's default.",
				NumberValidators: []validator.Number{validators.NumberAtLeast(0)},
			},
			field{
				Name: "connect_timeout", Kind: fieldNumber,
				Optional: true, Nullable: true,
				Description:      "Timeout in seconds for acquiring a connection from the pool. Unset uses aiohttp's default.",
				NumberValidators: []validator.Number{validators.NumberAtLeast(0)},
			},
			field{
				Name: "sock_connect_timeout", Kind: fieldNumber,
				Optional: true, Nullable: true,
				Description:      "Timeout in seconds for connecting to a peer for a new connection. Unset uses aiohttp's default.",
				NumberValidators: []validator.Number{validators.NumberAtLeast(0)},
			},
			field{
				Name: "sock_read_timeout", Kind: fieldNumber,
				Optional: true, Nullable: true,
				Description:      "Timeout in seconds for reading a portion of data from a peer. Unset uses aiohttp's default.",
				NumberValidators: []validator.Number{validators.NumberAtLeast(0)},
			},
		),
	}}
}
