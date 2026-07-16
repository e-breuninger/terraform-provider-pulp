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

var _ datasource.DataSource = &pulpDistributionsSource{}

func NewPulpDistributionsSource() datasource.DataSource {
	return &pulpDistributionsSource{}
}

type pulpDistributionsSource struct {
	client *client.PulpClient
}

// PulpDistributionsSourceModel is the top-level model for the pulp_distributions data source.
type PulpDistributionsSourceModel struct {
	ContentType   types.String `tfsdk:"content_type"`
	PluginName    types.String `tfsdk:"plugin_name"`
	Name          types.String `tfsdk:"name"`
	BasePath      types.String `tfsdk:"base_path"`
	Distributions types.List   `tfsdk:"distributions"`
}

func (r *pulpDistributionsSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_distributions"
}

func (r *pulpDistributionsSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all Pulp Distributions for a given content_type/plugin_name. Useful for discovering Distributions that were created dynamically, e.g. by a `container`/`pull-through` Distribution when an image is pulled through it.",
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
				MarkdownDescription: "Filter results to the Distribution with this exact `name`.",
			},
			"base_path": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Filter results to the Distribution with this exact `base_path`.",
			},
			"distributions": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The list of matching Distributions.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"pulp_href": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The `pulp_href` (used as the resource identifier).",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "A unique name for this Distribution.",
						},
						"base_path": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The base_path for this Distribution.",
						},
						"repository": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The name of the Repository that should be served at the base_path.",
						},
						"repository_version": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The version of the Repository.",
						},
						"allow_uploads": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "Whether to allow uploads to this Distribution.",
						},
						"remote": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The `pulp_href` of the Remote from which content should be pulled.",
						},
						"content_guard": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The `pulp_href` of the Content Guard used for this Distribution (if supported by the content_type/plugin_name).",
						},
						"namespace": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The namespace of this Distribution (if supported by the content_type/plugin_name).",
						},
						"private": schema.BoolAttribute{
							Computed:            true,
							MarkdownDescription: "If set to true, this disallows anonymous users to pull from this Distribution.",
						},
						"distributions": schema.ListAttribute{
							Computed:            true,
							ElementType:         types.StringType,
							MarkdownDescription: "List of Distributions that use this Distribution as a remote (if supported by the content_type/plugin_name).",
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

func (r *pulpDistributionsSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (r *pulpDistributionsSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config PulpDistributionsSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resPath := client.BuildResourcePath("distributions", config.ContentType.ValueString(), config.PluginName.ValueString())

	query := url.Values{}
	if !config.Name.IsNull() && !config.Name.IsUnknown() {
		query.Set("name", config.Name.ValueString())
	}
	if !config.BasePath.IsNull() && !config.BasePath.IsUnknown() {
		query.Set("base_path", config.BasePath.ValueString())
	}

	results, err := r.client.List(ctx, resPath, query)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list Distributions", err.Error())
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
		f := hydrateDistributionFields(ctx, data)
		obj, diags := types.ObjectValueFrom(ctx, distributionFieldsAttrTypes, f)
		resp.Diagnostics.Append(diags...)
		if diags.HasError() {
			continue
		}
		items = append(items, obj)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	list, diags := types.ListValue(types.ObjectType{AttrTypes: distributionFieldsAttrTypes}, items)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	config.Distributions = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
