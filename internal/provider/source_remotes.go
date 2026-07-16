// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/url"
	"sort"

	client "github.com/e-breuninger/terraform-provider-pulp/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &pulpRemotesSource{}

func NewPulpRemotesSource() datasource.DataSource {
	return &pulpRemotesSource{}
}

type pulpRemotesSource struct {
	client *client.PulpClient
}

// PulpRemotesSourceModel is the top-level model for the pulp_remotes data source.
type PulpRemotesSourceModel struct {
	ContentType types.String `tfsdk:"content_type"`
	PluginName  types.String `tfsdk:"plugin_name"`
	Name        types.String `tfsdk:"name"`
	Remotes     types.List   `tfsdk:"remotes"`
}

func (r *pulpRemotesSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_remotes"
}

func (r *pulpRemotesSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all Pulp Remotes for a given content_type/plugin_name.",
		Attributes: map[string]schema.Attribute{
			"content_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Content plugin type.",
			},
			"plugin_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Plugin sub-type if different from content_type.",
			},
			"name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Filter results to the Remote with this exact `name`.",
			},
			"remotes": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The list of matching Remotes.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"pulp_href": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The `pulp_href` (used as the resource identifier).",
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
				},
			},
		},
	}
}

func (r *pulpRemotesSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *pulpRemotesSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config PulpRemotesSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resPath := client.BuildResourcePath("remotes", config.ContentType.ValueString(), config.PluginName.ValueString())

	query := url.Values{}
	if !config.Name.IsNull() && !config.Name.IsUnknown() {
		query.Set("name", config.Name.ValueString())
	}

	results, err := r.client.List(ctx, resPath, query)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list Remotes", err.Error())
		return
	}

	// Sort by pulp_href for stable ordering across reads.
	sort.Slice(results, func(i, j int) bool {
		hrefI, _ := results[i]["pulp_href"].(string)
		hrefJ, _ := results[j]["pulp_href"].(string)
		return hrefI < hrefJ
	})

	items := make([]attr.Value, 0, len(results))
	for _, data := range results {
		f := hydrateRemoteFields(ctx, data)
		obj, diags := types.ObjectValueFrom(ctx, remoteFieldsAttrTypes, f)
		resp.Diagnostics.Append(diags...)
		if diags.HasError() {
			continue
		}
		items = append(items, obj)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	list, diags := types.ListValue(types.ObjectType{AttrTypes: remoteFieldsAttrTypes}, items)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.Remotes = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
