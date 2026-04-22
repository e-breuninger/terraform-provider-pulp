// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"

	"github.com/e-breuninger/terraform-provider-pulp/internal"
	client "github.com/e-breuninger/terraform-provider-pulp/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &pulpRemoteResource{}
var _ resource.ResourceWithImportState = &pulpRemoteResource{}

func NewPulpRemoteResource() resource.Resource {
	return &pulpRemoteResource{}
}

type pulpRemoteResource struct {
	client *client.PulpClient
}

type PulpRemoteModel struct {
	PulpHref      types.String `tfsdk:"pulp_href"`
	ContentType   types.String `tfsdk:"content_type"`
	PluginName    types.String `tfsdk:"plugin_name"`
	Name          types.String `tfsdk:"name"`
	Url           types.String `tfsdk:"url"`
	Policy        types.String `tfsdk:"policy"`
	TlsValidation types.Bool   `tfsdk:"tls_validation"`
	Username      types.String `tfsdk:"username"`
	Password      types.String `tfsdk:"password"`
	PulpLabels    types.Map    `tfsdk:"pulp_labels"`
}

func (r *pulpRemoteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_remote"
}

func (r *pulpRemoteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Pulp remote for any content type.",
		Attributes: map[string]schema.Attribute{
			"pulp_href": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The Pulp href (used as the resource identifier).",
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
				MarkdownDescription: "A unique name for this remote.",
			},
			"url": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The URL of an external content source.",
			},
			"policy": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Download policy: `immediate`, `on_demand`, or `streamed`.",
			},
			"tls_validation": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether TLS peer validation must be performed.",
			},
			"username": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Username for authentication when syncing.",
			},
			"password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Password for authentication when syncing.",
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

func (r *pulpRemoteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func buildRemoteBody(ctx context.Context, plan PulpRemoteModel) map[string]any {
	body := map[string]any{
		"name": plan.Name.ValueString(),
		"url":  plan.Url.ValueString(),
	}

	if !plan.Policy.IsNull() && !plan.Policy.IsUnknown() {
		body["policy"] = plan.Policy.ValueString()
	}
	if !plan.TlsValidation.IsNull() && !plan.TlsValidation.IsUnknown() {
		body["tls_validation"] = plan.TlsValidation.ValueBool()
	}
	if !plan.Username.IsNull() && !plan.Username.IsUnknown() {
		body["username"] = plan.Username.ValueString()
	}
	if !plan.Password.IsNull() && !plan.Password.IsUnknown() {
		body["password"] = plan.Password.ValueString()
	}
	if !plan.PulpLabels.IsNull() && !plan.PulpLabels.IsUnknown() {
		labels := make(map[string]string)
		plan.PulpLabels.ElementsAs(ctx, &labels, false)
		body["pulp_labels"] = labels
	}

	return body
}

func (r *pulpRemoteResource) resourcePath(plan PulpRemoteModel) string {
	return client.BuildResourcePath("remotes", plan.ContentType.ValueString(), plan.PluginName.ValueString())
}

// Hydrate the model from a Pulp API response map.
func hydrateRemoteModel(ctx context.Context, data map[string]any, model *PulpRemoteModel) {
	if v, ok := data["pulp_href"].(string); ok {
		model.PulpHref = types.StringValue(v)
	}
	if v, ok := data["name"].(string); ok {
		model.Name = types.StringValue(v)
	}
	if v, ok := data["url"].(string); ok {
		model.Url = types.StringValue(v)
	}
	if v, ok := data["policy"].(string); ok {
		model.Policy = types.StringValue(v)
	}
	if v, ok := data["tls_validation"].(bool); ok {
		model.TlsValidation = types.BoolValue(v)
	}
	if v, ok := data["username"].(string); ok && v != "" {
		model.Username = types.StringValue(v)
	}
	// password is write-only in Pulp, never returned
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

func (r *pulpRemoteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PulpRemoteModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := buildRemoteBody(ctx, plan)
	resPath := r.resourcePath(plan)

	result, err := r.client.Create(ctx, resPath, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create remote", err.Error())
		return
	}

	hydrateRemoteModel(ctx, result, &plan)

	// Default plugin_name to content_type if not set
	if plan.PluginName.IsNull() || plan.PluginName.IsUnknown() {
		plan.PluginName = plan.ContentType
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pulpRemoteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PulpRemoteModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.ReadByHref(ctx, state.PulpHref.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read remote", err.Error())
		return
	}
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	hydrateRemoteModel(ctx, result, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *pulpRemoteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PulpRemoteModel
	var state PulpRemoteModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := buildRemoteBody(ctx, plan)

	result, err := r.client.Update(ctx, state.PulpHref.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update remote", err.Error())
		return
	}

	// Preserve content_type and plugin_name from state (they force replace)
	plan.ContentType = state.ContentType
	plan.PluginName = state.PluginName

	hydrateRemoteModel(ctx, result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pulpRemoteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PulpRemoteModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(ctx, state.PulpHref.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete remote", err.Error())
		return
	}
}

func (r *pulpRemoteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := internal.ImportState(ctx, req, resp)

	contentType := parts[4]
	pluginName := parts[5]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("content_type"), contentType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("plugin_name"), pluginName)...)
}
