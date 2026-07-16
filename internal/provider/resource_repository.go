// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/e-breuninger/terraform-provider-pulp/internal"
	client "github.com/e-breuninger/terraform-provider-pulp/internal/client"
	validators "github.com/e-breuninger/terraform-provider-pulp/internal/validators"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &pulpRepositoryResource{}
var _ resource.ResourceWithImportState = &pulpRepositoryResource{}

func NewPulpRepositoryResource() resource.Resource {
	return &pulpRepositoryResource{}
}

type pulpRepositoryResource struct {
	client *client.PulpClient
}

type PulpRepositoryModel struct {
	PulpHref           types.String `tfsdk:"pulp_href"`
	ContentType        types.String `tfsdk:"content_type"`
	PluginName         types.String `tfsdk:"plugin_name"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	Remote             types.String `tfsdk:"remote"`
	PulpLabels         types.Map    `tfsdk:"pulp_labels"`
	RetainRepoVersions types.Int64  `tfsdk:"retain_repo_versions"`
}

func (r *pulpRepositoryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository"
}

func (r *pulpRepositoryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Pulp Repository for any content type.",
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
				MarkdownDescription: "A unique name for this Repository.",
			},
			"description": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "A description for this Repository.",
			},
			"remote": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "`pulp_href` of the Remote.",
				Validators: []validator.String{
					validators.PulpHrefValidator(),
				},
			},
			"pulp_labels": schema.MapAttribute{
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Key/value labels.",
			},
			"retain_repo_versions": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Maximum number of repository versions to retain. Omitting this attribute (or setting it to `null`) keeps all versions.",
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
		},
	}
}

func (r *pulpRepositoryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Helper: build the body map from the plan, skipping null/unknown values.
func buildRepositoryBody(ctx context.Context, plan PulpRepositoryModel) map[string]any {
	body := map[string]any{
		"name":        plan.Name.ValueString(),
		"description": plan.Description.ValueString(),
	}

	// Both remote and retain_repo_versions are nullable in Pulp.
	internal.SetStrField(body, "remote", plan.Remote, true)
	internal.SetIntField(body, "retain_repo_versions", plan.RetainRepoVersions, true)

	if labels := internal.LabelsToMap(ctx, plan.PulpLabels); labels != nil {
		body["pulp_labels"] = labels
	}

	return body
}

func (r *pulpRepositoryResource) resourcePath(plan PulpRepositoryModel) string {
	return client.BuildResourcePath("repositories", plan.ContentType.ValueString(), plan.PluginName.ValueString())
}

// Hydrate the model from a Pulp API response map.
func hydrateRepositoryModel(ctx context.Context, data map[string]any, model *PulpRepositoryModel) {
	model.PulpHref = internal.StrOrNull(data, "pulp_href")
	model.Name = internal.StrOrNull(data, "name")
	model.Description = internal.StrOrNull(data, "description")
	model.Remote = internal.StrOrNull(data, "remote")
	model.RetainRepoVersions = internal.IntOrNull(data, "retain_repo_versions")
	model.PulpLabels = internal.LabelsOrNull(ctx, data)
}

func (r *pulpRepositoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PulpRepositoryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := buildRepositoryBody(ctx, plan)
	resPath := r.resourcePath(plan)

	result, err := r.client.Create(ctx, resPath, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Repository", err.Error())
		return
	}

	hydrateRepositoryModel(ctx, result, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pulpRepositoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PulpRepositoryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.ReadByHref(ctx, state.PulpHref.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Repository", err.Error())
		return
	}
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	hydrateRepositoryModel(ctx, result, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *pulpRepositoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PulpRepositoryModel
	var state PulpRepositoryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := buildRepositoryBody(ctx, plan)

	result, err := r.client.Update(ctx, state.PulpHref.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update Repository", err.Error())
		return
	}

	// Preserve content_type and plugin_name from state (they force replace)
	plan.ContentType = state.ContentType
	plan.PluginName = state.PluginName

	hydrateRepositoryModel(ctx, result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pulpRepositoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PulpRepositoryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(ctx, state.PulpHref.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete Repository", err.Error())
		return
	}
}

func (r *pulpRepositoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := internal.ImportState(ctx, req, resp)

	contentType := parts[4]
	pluginName := parts[5]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("content_type"), contentType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("plugin_name"), pluginName)...)
}
