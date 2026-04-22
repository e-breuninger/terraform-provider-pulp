// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"math/big"
	"regexp"

	"github.com/e-breuninger/terraform-provider-pulp/internal"
	"github.com/e-breuninger/terraform-provider-pulp/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &pulpUserRoleResource{}
var _ resource.ResourceWithImportState = &pulpUserRoleResource{}

func NewPulpUserRoleResource() resource.Resource {
	return &pulpUserRoleResource{}
}

type pulpUserRoleResource struct {
	client *client.PulpClient
}

type PulpUserRoleModel struct {
	PulpHref         types.String `tfsdk:"pulp_href"`
	UserID           types.Number `tfsdk:"user_id"`
	Role             types.String `tfsdk:"role"`
	ContentObject    types.String `tfsdk:"content_object"`
	ContentObjectPrn types.String `tfsdk:"content_object_prn"`
	Domain           types.String `tfsdk:"domain"`
}

func (r *pulpUserRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_role"
}

func (r *pulpUserRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Pulp UserRole.",
		Attributes: map[string]schema.Attribute{
			"pulp_href": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The Pulp href (used as the resource identifier).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user_id": schema.NumberAttribute{
				Required:            true,
				MarkdownDescription: "The user that gets this UserRole.",
			},
			"role": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The role to assign to the user.",
			},
			"content_object": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The href of the object this role applies to.",
			},
			"content_object_prn": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "The PRN of the content object for which role permissions should be asserted. " +
					"If set to 'null', permissions will act on either domain or model-level.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Domain this role should be applied on, mutually exclusive with content_object.",
			},
		},
	}
}

func (r *pulpUserRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func buildUserRoleBody(_ context.Context, plan PulpUserRoleModel) map[string]any {
	body := map[string]any{
		"role":               plan.Role.ValueString(),
		"content_object":     nil,
		"content_object_prn": nil,
	}

	if !plan.ContentObject.IsNull() && !plan.ContentObject.IsUnknown() {
		body["content_object"] = plan.ContentObject.ValueString()
	}
	if !plan.ContentObjectPrn.IsNull() && !plan.ContentObjectPrn.IsUnknown() {
		body["content_object_prn"] = plan.ContentObjectPrn.ValueString()
	}
	if !plan.Domain.IsNull() && !plan.Domain.IsUnknown() {
		body["domain"] = plan.Domain.ValueString()
	}

	return body
}

func (r *pulpUserRoleResource) resourcePath(plan PulpUserRoleModel) string {
	return client.BuildResourcePath("users", plan.UserID.String(), "roles")
}

var userRoleHrefRegex = regexp.MustCompile(`/users/(\d+)/roles/`)

func hydrateUserRoleModel(ctx context.Context, data map[string]any, model *PulpUserRoleModel) {
	tflog.Debug(ctx, "Hydrating user_role model", map[string]any{
		"data": fmt.Sprintf("%+v", data),
	})
	if v, ok := data["pulp_href"].(string); ok {
		model.PulpHref = types.StringValue(v)

		// Extract user_id from the href
		if matches := userRoleHrefRegex.FindStringSubmatch(v); len(matches) == 2 {
			if n, ok := new(big.Float).SetString(matches[1]); ok {
				model.UserID = types.NumberValue(n)
			}
		}
	}
	if v, ok := data["role"].(string); ok {
		model.Role = types.StringValue(v)
	}
	if v, ok := data["content_object"].(string); ok && v != "" {
		model.ContentObject = types.StringValue(v)
	} else {
		model.ContentObject = types.StringNull()
	}
	if v, ok := data["content_object_prn"].(string); ok && v != "" {
		model.ContentObjectPrn = types.StringValue(v)
	} else {
		model.ContentObjectPrn = types.StringNull()
	}

	if v, ok := data["domain"].(string); ok {
		model.Domain = types.StringValue(v)
	}
}

func (r *pulpUserRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PulpUserRoleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := buildUserRoleBody(ctx, plan)
	resPath := r.resourcePath(plan)

	result, err := r.client.Create(ctx, resPath, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create UserRole", err.Error())
		return
	}

	hydrateUserRoleModel(ctx, result, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pulpUserRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PulpUserRoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.ReadByHref(ctx, state.PulpHref.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read UserRole", err.Error())
		return
	}
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	hydrateUserRoleModel(ctx, result, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update patches the UserRole by Deleting and Re-creating the UserRole
func (r *pulpUserRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PulpUserRoleModel
	var state PulpUserRoleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the UserRole
	if err := r.client.Delete(ctx, state.PulpHref.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete old UserRole during update", err.Error())
		return
	}

	// Create a new UserRole
	body := buildUserRoleBody(ctx, plan)
	resPath := r.resourcePath(plan)

	result, err := r.client.Create(ctx, resPath, body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create new UserRole during update",
			fmt.Sprintf("The old UserRole was deleted but the new one could not be created. "+
				"You may need to re-import or recreate the resource. Error: %s", err.Error()),
		)
		return
	}

	hydrateUserRoleModel(ctx, result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pulpUserRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PulpUserRoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(ctx, state.PulpHref.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete UserRole", err.Error())
		return
	}
}

func (r *pulpUserRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	internal.ImportState(ctx, req, resp)
}
