// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	internal "github.com/e-breuninger/terraform-provider-pulp/internal"
	"github.com/e-breuninger/terraform-provider-pulp/internal/client"
	"github.com/e-breuninger/terraform-provider-pulp/internal/modifiers"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &pulpObjectRoleResource{}
	_ resource.ResourceWithImportState = &pulpObjectRoleResource{}
)

func NewPulpObjectRoleResource() resource.Resource {
	return &pulpObjectRoleResource{}
}

type pulpObjectRoleResource struct {
	client *client.PulpClient
}

// PulpObjectRoleModel represents a role assignment (users+groups -> role) on a generic Pulp object.
// Since Pulp does not return a dedicated pulp_href for a role assignment, we synthesize a composite
// ID from the Object href and the role name.
type PulpObjectRoleModel struct {
	PulpHref types.String `tfsdk:"pulp_href"`
	Users    types.List   `tfsdk:"users"`
	Groups   types.List   `tfsdk:"groups"`
	Role     types.String `tfsdk:"role"`
}

func (r *pulpObjectRoleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_object_role"
}

func (r *pulpObjectRoleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a role assignment on Pulp objects.",
		Attributes: map[string]schema.Attribute{
			"pulp_href": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The full pulp_href of the Object that gets this role.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"users": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					modifiers.OrderListModifier{},
				},
				MarkdownDescription: "List of usernames to assign the role to.",
			},
			"groups": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					modifiers.OrderListModifier{},
				},
				MarkdownDescription: "List of group names to assign the role to.",
			},
			"role": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The role to assign.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *pulpObjectRoleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// readRoleAssignment fetches current users/groups for a given role on an Object.
// Returns (users, groups, found).
func (r *pulpObjectRoleResource) readRoleAssignment(
	ctx context.Context, pulpHref, role string,
) (users, groups []string, found bool, err error) {
	result, statusCode, err := r.client.ListHrefAction(ctx, pulpHref, "list_roles")
	if err != nil {
		return nil, nil, false, err
	}
	if statusCode != http.StatusOK {
		return nil, nil, false, fmt.Errorf("list_roles failed with status %d", statusCode)
	}
	rolesRaw, ok := result["roles"].([]any)
	if !ok {
		return nil, nil, false, nil
	}
	for _, entry := range rolesRaw {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		name, _ := m["role"].(string)
		if name != role {
			continue
		}
		found = true
		if us, ok := m["users"].([]any); ok {
			for _, u := range us {
				if s, ok := u.(string); ok {
					users = append(users, s)
				}
			}
		}
		if gs, ok := m["groups"].([]any); ok {
			for _, g := range gs {
				if s, ok := g.(string); ok {
					groups = append(groups, s)
				}
			}
		}
		break
	}
	return users, groups, found, nil
}

func (r *pulpObjectRoleResource) addRole(
	ctx context.Context, pulpHref, role string, users, groups []string,
) error {
	if len(users) == 0 && len(groups) == 0 {
		return nil
	}
	body := map[string]any{
		"role":   role,
		"users":  users,
		"groups": groups,
	}
	if users == nil {
		body["users"] = []string{}
	}
	if groups == nil {
		body["groups"] = []string{}
	}
	tflog.Error(ctx, "Adding object role", map[string]any{"href": pulpHref, "body": body})
	resp, statusCode, err := r.client.CallHrefAction(ctx, pulpHref, "add_role", body)
	if (statusCode != http.StatusOK && statusCode != http.StatusCreated) || err != nil {
		return fmt.Errorf("add_role failed with status %d\nbody: %v\nerror: %v\nresponse: %v", statusCode, body, err, resp)
	}
	return nil
}

func (r *pulpObjectRoleResource) removeRole(
	ctx context.Context, pulpHref, role string, users, groups []string,
) error {
	if len(users) == 0 && len(groups) == 0 {
		return nil
	}
	body := map[string]any{
		"role":   role,
		"users":  users,
		"groups": groups,
	}
	if users == nil {
		body["users"] = []string{}
	}
	if groups == nil {
		body["groups"] = []string{}
	}
	tflog.Error(ctx, "Removing object role", map[string]any{"href": pulpHref, "body": body})
	_, statusCode, err := r.client.CallHrefAction(ctx, pulpHref, "remove_role", body)
	if err != nil {
		return err
	}
	if statusCode != http.StatusOK {
		return fmt.Errorf("remove_role failed with status %d", statusCode)
	}
	return nil
}

func (r *pulpObjectRoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PulpObjectRoleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pulpHref := plan.PulpHref.ValueString()
	role := plan.Role.ValueString()

	users, err := internal.ListToStrings(ctx, plan.Users)
	if err != nil {
		resp.Diagnostics.AddError("Invalid users list", err.Error())
		return
	}
	groups, err := internal.ListToStrings(ctx, plan.Groups)
	if err != nil {
		resp.Diagnostics.AddError("Invalid groups list", err.Error())
		return
	}

	// Reconcile against the current state (role may already exist with a subset of the desired
	// users/groups, e.g. if this resource is taking over management of an existing assignment).
	curUsers, curGroups, _, err := r.readRoleAssignment(ctx, pulpHref, role)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list Object roles", err.Error())
		return
	}
	addUsers, _ := internal.Diff(users, curUsers)
	addGroups, _ := internal.Diff(groups, curGroups)

	if err := r.addRole(ctx, pulpHref, role, addUsers, addGroups); err != nil {
		resp.Diagnostics.AddError("Failed to add role on Object", err.Error())
		return
	}

	// Refresh from server to populate computed values.
	finalUsers, finalGroups, found, err := r.readRoleAssignment(ctx, pulpHref, role)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back Object role", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Role assignment not found after create",
			fmt.Sprintf("role %q on %s was not present after add_role call", role, pulpHref))
		return
	}

	plan.Users = internal.StringsToList(internal.Intersect(users, finalUsers))
	plan.Groups = internal.StringsToList(internal.Intersect(groups, finalGroups))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pulpObjectRoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PulpObjectRoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pulpHref := state.PulpHref.ValueString()
	role := state.Role.ValueString()

	users, groups, found, err := r.readRoleAssignment(ctx, pulpHref, role)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Object role", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	// We only manage the users/groups that were declared in config. However, since we can't know
	// that here without comparing to a plan, we reflect the full server-side state. Drift on other
	// users/groups under the same role will show up — which is usually what Terraform users want.
	state.Users = internal.StringsToList(users)
	state.Groups = internal.StringsToList(groups)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update diffs users/groups and calls add_role / remove_role as needed.
// pulp_href and role are RequiresReplace, so they won't change here.
func (r *pulpObjectRoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state PulpObjectRoleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pulpHref := plan.PulpHref.ValueString()
	role := plan.Role.ValueString()

	desiredUsers, err := internal.ListToStrings(ctx, plan.Users)
	if err != nil {
		resp.Diagnostics.AddError("Invalid users list", err.Error())
		return
	}
	desiredGroups, err := internal.ListToStrings(ctx, plan.Groups)
	if err != nil {
		resp.Diagnostics.AddError("Invalid groups list", err.Error())
		return
	}

	// Pull current server-side state to diff against (more reliable than Terraform state).
	curUsers, curGroups, _, err := r.readRoleAssignment(ctx, pulpHref, role)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list Object roles", err.Error())
		return
	}

	addUsers, removeUsers := internal.Diff(desiredUsers, curUsers)
	addGroups, removeGroups := internal.Diff(desiredGroups, curGroups)

	if err := r.addRole(ctx, pulpHref, role, addUsers, addGroups); err != nil {
		resp.Diagnostics.AddError("Failed to add role members", err.Error())
		return
	}
	if err := r.removeRole(ctx, pulpHref, role, removeUsers, removeGroups); err != nil {
		resp.Diagnostics.AddError("Failed to remove role members", err.Error())
		return
	}

	finalUsers, finalGroups, found, err := r.readRoleAssignment(ctx, pulpHref, role)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back Object role", err.Error())
		return
	}
	if !found {
		// All members removed -> resource no longer exists on server.
		resp.State.RemoveResource(ctx)
		return
	}

	plan.Users = internal.StringsToList(finalUsers)
	plan.Groups = internal.StringsToList(finalGroups)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pulpObjectRoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PulpObjectRoleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	pulpHref := state.PulpHref.ValueString()
	role := state.Role.ValueString()

	users, err := internal.ListToStrings(ctx, state.Users)
	if err != nil {
		resp.Diagnostics.AddError("Invalid users list", err.Error())
		return
	}
	groups, err := internal.ListToStrings(ctx, state.Groups)
	if err != nil {
		resp.Diagnostics.AddError("Invalid groups list", err.Error())
		return
	}

	if err := r.removeRole(ctx, pulpHref, role, users, groups); err != nil {
		resp.Diagnostics.AddError("Failed to remove Object role", err.Error())
		return
	}
}

func (r *pulpObjectRoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "|", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid composite ID", fmt.Sprintf("Composite ID is %q, expected `<pulp_href>|<role>`", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("pulp_href"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role"), parts[1])...)
}
