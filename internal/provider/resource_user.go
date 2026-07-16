// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/e-breuninger/terraform-provider-pulp/internal"
	"github.com/e-breuninger/terraform-provider-pulp/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/numberplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &pulpUserResource{}
var _ resource.ResourceWithImportState = &pulpUserResource{}

func NewPulpUserResource() resource.Resource {
	return &pulpUserResource{}
}

type pulpUserResource struct {
	client *client.PulpClient
}

type PulpUserModel struct {
	PulpHref  types.String `tfsdk:"pulp_href"`
	ID        types.Number `tfsdk:"id"`
	Username  types.String `tfsdk:"username"`
	Password  types.String `tfsdk:"password"`
	FirstName types.String `tfsdk:"first_name"`
	LastName  types.String `tfsdk:"last_name"`
	Email     types.String `tfsdk:"email"`
	IsStaff   types.Bool   `tfsdk:"is_staff"`
	IsActive  types.Bool   `tfsdk:"is_active"`
}

func (r *pulpUserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *pulpUserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Pulp User.",
		Attributes: map[string]schema.Attribute{
			"pulp_href": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The `pulp_href` (used as the resource identifier).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"id": schema.NumberAttribute{
				Computed:            true,
				MarkdownDescription: "The Pulp user ID.",
				PlanModifiers: []planmodifier.Number{
					numberplanmodifier.UseStateForUnknown(),
				},
			},
			"username": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "A unique username for this User. 150 characters or fewer. Letters, digits and `@.+-_` only.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^[A-Za-z0-9@.+_-]{1,150}$`),
						"must be 150 characters or fewer and contain only letters, digits, and @.+-_"),
				},
			},
			"password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "The password for this User. Pulp allows empty passwords but they are not recommended.",
			},
			"first_name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The first name of this User.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The last name of this User.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"email": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The email address of this User.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_staff": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether this User can log into the admin site.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"is_active": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether this User account is active.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *pulpUserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func buildUserBody(_ context.Context, plan PulpUserModel) map[string]any {
	body := map[string]any{
		"username": plan.Username.ValueString(),
	}

	// password is nullable in Pulp; the rest are not.
	internal.SetStrField(body, "password", plan.Password, true)
	internal.SetStrField(body, "first_name", plan.FirstName, false)
	internal.SetStrField(body, "last_name", plan.LastName, false)
	internal.SetStrField(body, "email", plan.Email, false)
	internal.SetBoolField(body, "is_staff", plan.IsStaff, false)
	internal.SetBoolField(body, "is_active", plan.IsActive, false)

	return body
}

func (r *pulpUserResource) resourcePath() string {
	return "users"
}

func hydrateUserModel(ctx context.Context, data map[string]any, model *PulpUserModel) {
	tflog.Debug(ctx, "Hydrating user model", map[string]any{
		"data": fmt.Sprintf("%+v", data),
	})
	model.PulpHref = internal.StrOrNull(data, "pulp_href")
	model.ID = internal.NumberOrNull(data, "id")
	model.Username = internal.StrOrNull(data, "username")
	model.FirstName = internal.StrOrNull(data, "first_name")
	model.LastName = internal.StrOrNull(data, "last_name")
	model.Email = internal.StrOrNull(data, "email")
	model.IsStaff = internal.BoolOrNull(data, "is_staff")
	model.IsActive = internal.BoolOrNull(data, "is_active")
}

func (r *pulpUserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PulpUserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := buildUserBody(ctx, plan)
	resPath := r.resourcePath()

	result, err := r.client.Create(ctx, resPath, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create User", err.Error())
		return
	}

	hydrateUserModel(ctx, result, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pulpUserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PulpUserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.ReadByHref(ctx, state.PulpHref.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read User", err.Error())
		return
	}
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	hydrateUserModel(ctx, result, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *pulpUserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PulpUserModel
	var state PulpUserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := buildUserBody(ctx, plan)

	result, err := r.client.Update(ctx, state.PulpHref.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update User", err.Error())
		return
	}

	hydrateUserModel(ctx, result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pulpUserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PulpUserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(ctx, state.PulpHref.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete User", err.Error())
		return
	}
}

func (r *pulpUserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	internal.ImportState(ctx, req, resp)
}
