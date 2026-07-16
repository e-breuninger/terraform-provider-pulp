// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	client "github.com/e-breuninger/terraform-provider-pulp/internal/client"
	validators "github.com/e-breuninger/terraform-provider-pulp/internal/validators"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &pulpDistributionSource{}

func NewPulpDistributionSource() datasource.DataSource {
	return &pulpDistributionSource{}
}

type pulpDistributionSource struct {
	client *client.PulpClient
}

type PulpDistributionSourceModel struct {
	PulpHref          types.String `tfsdk:"pulp_href"`
	ContentType       types.String `tfsdk:"content_type"`
	PluginName        types.String `tfsdk:"plugin_name"`
	Name              types.String `tfsdk:"name"`
	BasePath          types.String `tfsdk:"base_path"`
	Repository        types.String `tfsdk:"repository"`
	RepositoryVersion types.String `tfsdk:"repository_version"`
	AllowUploads      types.Bool   `tfsdk:"allow_uploads"`
	Remote            types.String `tfsdk:"remote"`
	ContentGuard      types.String `tfsdk:"content_guard"`
	Namespace         types.String `tfsdk:"namespace"`
	Private           types.Bool   `tfsdk:"private"`
	Distributions     types.List   `tfsdk:"distributions"`
	PulpLabels        types.Map    `tfsdk:"pulp_labels"`
}

func (r *pulpDistributionSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_distribution"
}

func (r *pulpDistributionSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Pulp Distribution for any content type.",
		Attributes: map[string]schema.Attribute{
			"pulp_href": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The `pulp_href` (used as the resource identifier).",
			},
			"content_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Content plugin type.",
			},
			"plugin_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Plugin sub-type if different from content_type.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "A unique name for this Distribution.",
			},
			"base_path": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The base_path for this Distribution.",
			},
			"repository": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The name of the Repository that should be served at the base_path.",
			},
			"repository_version": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The version of the Repository.",
			},
			"allow_uploads": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether to allow uploads to this Distribution.",
			},
			"remote": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The `pulp_href` of the Remote from which content should be pulled.",
				Validators: []validator.String{
					validators.PulpHrefValidator(),
				},
			},
			"content_guard": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The `pulp_href` of the Content Guard to use for this Distribution (if supported by the content_type/plugin_name).",
				Validators: []validator.String{
					validators.PulpHrefValidator(),
				},
			},
			"namespace": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The namespace of this Distribution (if supported by the content_type/plugin_name).",
			},
			"distributions": schema.ListAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of Distributions that use this Distribution as a remote (if supported by the content_type/plugin_name).",
			},
			"private": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "If set to true, this disallows anonymous users to pull from this Distribution.",
			},
			"pulp_labels": schema.MapAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Key/value labels.",
			},
		},
	}
}

func (r *pulpDistributionSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func hydrateDistributionSourceModel(ctx context.Context, data map[string]any, model *PulpDistributionSourceModel) {
	tflog.Debug(ctx, "Hydrating distribution model", map[string]any{
		"data": fmt.Sprintf("%+v", data),
	})
	f := hydrateDistributionFields(ctx, data)
	model.PulpHref = f.PulpHref
	model.Name = f.Name
	model.BasePath = f.BasePath
	model.Repository = f.Repository
	model.RepositoryVersion = f.RepositoryVersion
	model.AllowUploads = f.AllowUploads
	model.Remote = f.Remote
	model.ContentGuard = f.ContentGuard
	model.Namespace = f.Namespace
	model.Private = f.Private
	model.Distributions = f.Distributions
	model.PulpLabels = f.PulpLabels
}

func (r *pulpDistributionSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state PulpDistributionSourceModel
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.ReadByHref(ctx, state.PulpHref.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Distribution", err.Error())
		return
	}
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	hydrateDistributionSourceModel(ctx, result, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
