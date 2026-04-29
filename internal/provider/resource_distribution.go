// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	internal "github.com/e-breuninger/terraform-provider-pulp/internal"
	client "github.com/e-breuninger/terraform-provider-pulp/internal/client"
	validators "github.com/e-breuninger/terraform-provider-pulp/internal/validators"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &pulpDistributionResource{}
var _ resource.ResourceWithImportState = &pulpDistributionResource{}

func NewPulpDistributionResource() resource.Resource {
	return &pulpDistributionResource{}
}

type pulpDistributionResource struct {
	client *client.PulpClient
}

type PulpDistributionModel struct {
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

func (r *pulpDistributionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_distribution"
}

func (r *pulpDistributionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Pulp Distribution for any content type.",
		Attributes: map[string]schema.Attribute{
			"pulp_href": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The `pulp_href` (used as the resource identifier).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"content_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Content plugin type.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(
						"ansible",
						"container",
						"core",
						"deb",
						"file",
						"hugging_face",
						"maven",
						"npm",
						"ostree",
						"python",
						"rpm",
					),
				},
			},
			"plugin_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Plugin sub-type if different from content_type.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"ansible",
						"container",
						"pull-through",
						"artifacts",
						"openpgp",
						"apt",
						"file",
						"hugging-face",
						"maven",
						"npm",
						"ostree",
						"pypi",
						"rpm",
					),
				},
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"distributions": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						validators.PulpHrefValidator(),
					),
				},
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

func (r *pulpDistributionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Map of supported features by content_type/plugin_name.
var supportedFeatures = map[string]map[string]bool{
	"file/file":              {"content_guard": true},
	"python/pypi":            {"content_guard": true},
	"npm/npm":                {"content_guard": true},
	"container/container":    {"content_guard": true},
	"container/pull-through": {"content_guard": true, "private": true, "namespace": true, "distributions": true},
	"rpm/rpm":                {"content_guard": true},
	"deb/apt":                {"content_guard": true},
	"ansible/ansible":        {"content_guard": true},
	"maven/maven":            {"content_guard": true},
	"ostree/ostree":          {"content_guard": true},
	"core/artifacts":         {"content_guard": true},
	"core/openpgp":           {"content_guard": true},
}

func supportsFeature(contentType, pluginName string, feature string) bool {
	return supportedFeatures[fmt.Sprintf("%s/%s", contentType, pluginName)][feature]
}

// Helper: build the body map from the plan, skipping null/unknown values.
func buildDistributionBody(ctx context.Context, plan PulpDistributionModel) map[string]any {
	body := map[string]any{
		"name":      plan.Name.ValueString(),
		"base_path": plan.BasePath.ValueString(),
	}

	if !plan.Repository.IsNull() && !plan.Repository.IsUnknown() {
		body["repository"] = plan.Repository.ValueString()
	}
	if !plan.RepositoryVersion.IsNull() && !plan.RepositoryVersion.IsUnknown() {
		body["repository_version"] = plan.RepositoryVersion.ValueString()
	}
	if !plan.AllowUploads.IsNull() && !plan.AllowUploads.IsUnknown() {
		body["allow_uploads"] = plan.AllowUploads.ValueBool()
	}
	if !plan.Remote.IsNull() && !plan.Remote.IsUnknown() {
		body["remote"] = plan.Remote.ValueString()
	}

	// Convert pulp_labels from types.Map to map[string]string
	if !plan.PulpLabels.IsNull() && !plan.PulpLabels.IsUnknown() {
		labels := make(map[string]string)
		plan.PulpLabels.ElementsAs(ctx, &labels, false)
		body["pulp_labels"] = labels
	}

	// Not every distribution has a content guard
	if supportsFeature(plan.ContentType.ValueString(), plan.PluginName.ValueString(), "content_guard") {
		if !plan.ContentGuard.IsNull() && !plan.ContentGuard.IsUnknown() {
			body["content_guard"] = plan.ContentGuard.ValueString()
		}
	}

	// Not every distribution has a distributions attribute
	if supportsFeature(plan.ContentType.ValueString(), plan.PluginName.ValueString(), "distributions") {
		if !plan.Distributions.IsNull() && !plan.Distributions.IsUnknown() {
			var distList []string
			plan.Distributions.ElementsAs(ctx, &distList, false)
			body["distributions"] = distList
		}
	}

	// Not every distribution supports the private flag
	if supportsFeature(plan.ContentType.ValueString(), plan.PluginName.ValueString(), "private") {
		if !plan.Private.IsNull() && !plan.Private.IsUnknown() {
			body["private"] = plan.Private.ValueBool()
		}
	}

	return body
}

func (r *pulpDistributionResource) resourcePath(plan PulpDistributionModel) string {
	return client.BuildResourcePath("distributions", plan.ContentType.ValueString(), plan.PluginName.ValueString())
}

// Hydrate the model from a Pulp API response map.
func hydrateDistributionModel(ctx context.Context, data map[string]any, model *PulpDistributionModel) {
	tflog.Debug(ctx, "Hydrating distribution model", map[string]any{
		"data": fmt.Sprintf("%+v", data),
	})
	if v, ok := data["pulp_href"].(string); ok {
		model.PulpHref = types.StringValue(v)
	}
	if v, ok := data["name"].(string); ok {
		model.Name = types.StringValue(v)
	}
	if v, ok := data["base_path"].(string); ok {
		model.BasePath = types.StringValue(v)
	}
	if v, ok := data["repository"].(string); ok && v != "" {
		model.Repository = types.StringValue(v)
	} else {
		model.Repository = types.StringNull()
	}
	if v, ok := data["repository_version"].(string); ok && v != "" {
		model.RepositoryVersion = types.StringValue(v)
	} else {
		model.RepositoryVersion = types.StringNull()
	}

	if v, ok := data["allow_uploads"].(bool); ok {
		model.AllowUploads = types.BoolValue(v)
	} else {
		model.AllowUploads = types.BoolNull()
	}

	if v, ok := data["remote"].(string); ok && v != "" {
		model.Remote = types.StringValue(v)
	} else {
		model.Remote = types.StringNull()
	}

	if v, ok := data["content_guard"].(string); ok && v != "" {
		model.ContentGuard = types.StringValue(v)
	} else {
		model.ContentGuard = types.StringNull()
	}

	if v, ok := data["namespace"].(string); ok && v != "" {
		model.Namespace = types.StringValue(v)
	} else {
		model.Namespace = types.StringNull()
	}

	if _, ok := data["distributions"].([]any); ok {
		model.Distributions = *internal.StringList(ctx, data, "distributions")
	} else {
		model.Distributions = types.ListNull(types.StringType)
	}

	if v, ok := data["private"].(bool); ok {
		model.Private = types.BoolValue(v)
	} else {
		model.Private = types.BoolNull()
	}

	// Convert pulp_labels from map[string]any to types.Map
	if v, ok := data["pulp_labels"].(map[string]any); ok {
		elems := make(map[string]types.String)
		for k, val := range v {
			if s, ok := val.(string); ok {
				elems[k] = types.StringValue(s)
			}
		}
		// Convert to types.Map
		labels := make(map[string]string)
		for k, val := range v {
			if s, ok := val.(string); ok {
				labels[k] = s
			}
		}
		mapVal, _ := types.MapValueFrom(ctx, types.StringType, labels)
		model.PulpLabels = mapVal
	}
}

func (r *pulpDistributionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PulpDistributionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := buildDistributionBody(ctx, plan)
	resPath := r.resourcePath(plan)

	result, err := r.client.Create(ctx, resPath, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Distribution", err.Error())
		return
	}

	hydrateDistributionModel(ctx, result, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pulpDistributionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PulpDistributionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
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

	hydrateDistributionModel(ctx, result, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *pulpDistributionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PulpDistributionModel
	var state PulpDistributionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := buildDistributionBody(ctx, plan)

	result, err := r.client.Update(ctx, state.PulpHref.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update Distribution", err.Error())
		return
	}

	// Preserve content_type and plugin_name from state (they force replace)
	plan.ContentType = state.ContentType
	plan.PluginName = state.PluginName

	hydrateDistributionModel(ctx, result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pulpDistributionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PulpDistributionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(ctx, state.PulpHref.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete Distribution", err.Error())
		return
	}
}

func (r *pulpDistributionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := internal.ImportState(ctx, req, resp)

	contentType := parts[4]
	pluginName := parts[5]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("content_type"), contentType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("plugin_name"), pluginName)...)
}
