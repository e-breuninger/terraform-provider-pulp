// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	client "github.com/e-breuninger/terraform-provider-pulp/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &pulpRemoteSource{}

func NewPulpRemoteSource() datasource.DataSource {
	return &pulpRemoteSource{}
}

type pulpRemoteSource struct {
	client *client.PulpClient
}

type PulpRemoteSourceModel struct {
	PulpHref      types.String `tfsdk:"pulp_href"`
	ContentType   types.String `tfsdk:"content_type"`
	PluginName    types.String `tfsdk:"plugin_name"`
	Name          types.String `tfsdk:"name"`
	Url           types.String `tfsdk:"url"`
	Policy        types.String `tfsdk:"policy"`
	TlsValidation types.Bool   `tfsdk:"tls_validation"`
	Username      types.String `tfsdk:"username"`
	PulpLabels    types.Map    `tfsdk:"pulp_labels"`
}

func (r *pulpRemoteSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_remote"
}

func (r *pulpRemoteSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a Pulp Remote for any content type.",
		Attributes: map[string]schema.Attribute{
			"pulp_href": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The `pulp_href` (used as the resource identifier).",
			},
			"content_type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Content plugin type.",
			},
			"plugin_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Plugin sub-type if different from content_type.",
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "A unique name for this Remote.",
			},
			"url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The URL of an external content source.",
			},
			"policy": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Download policy: `immediate`, `on_demand`, or `streamed`.",
			},
			"tls_validation": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether TLS peer validation must be performed.",
			},
			"username": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Username for authentication when syncing.",
			},
			"pulp_labels": schema.MapAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Key/value labels.",
			},
		},
	}
}

func (r *pulpRemoteSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.PulpClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type",
			fmt.Sprintf("Expected *client.PulpClient, got %T", req.ProviderData))
		return
	}
	r.client = c
}

// Hydrate the model from a Pulp API response map.
func hydrateRemoteSourceModel(ctx context.Context, data map[string]any, model *PulpRemoteSourceModel) {
	f := hydrateRemoteFields(ctx, data)
	model.PulpHref = f.PulpHref
	model.Name = f.Name
	model.Url = f.Url
	model.Policy = f.Policy
	model.TlsValidation = f.TlsValidation
	model.Username = f.Username
	model.PulpLabels = f.PulpLabels
}

func (r *pulpRemoteSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config PulpRemoteSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.ReadByHref(ctx, config.PulpHref.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Remote", err.Error())
		return
	}
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	hydrateRemoteSourceModel(ctx, result, &config)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
