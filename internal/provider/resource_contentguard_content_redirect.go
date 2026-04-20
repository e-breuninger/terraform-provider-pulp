// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	client "github.com/e-breuninger/terraform-provider-pulp/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &pulpContentGuardResource{}
var _ resource.ResourceWithImportState = &pulpContentGuardResource{}

func NewPulpContentGuardResource() resource.Resource {
	return &pulpContentGuardResource{}
}

type pulpContentGuardResource struct {
	client *client.PulpClient
}

type PulpContentGuardModel struct {
	PulpHref    types.String `tfsdk:"pulp_href"`
	ContentType types.String `tfsdk:"content_type"`
	PluginName  types.String `tfsdk:"plugin_name"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	// Roles       types.List   `tfsdk:"roles"`

	// Contentguards: X509 exclusive
	CaCertificate types.String `tfsdk:"ca_certificate"`

	// Contentguards: Composite exclusive
	Guards types.List `tfsdk:"guards"`

	// Contentguards: Header exclusive
	HeaderName  types.String `tfsdk:"header_name"`
	HeaderValue types.String `tfsdk:"header_value"`
	JqFilter    types.String `tfsdk:"jq_filter"`

	// Contentguards: Rbac exclusive
	Users  types.List `tfsdk:"users"`
	Groups types.List `tfsdk:"groups"`
}

func (r *pulpContentGuardResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_contentguard"
}

func (r *pulpContentGuardResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a Pulp ContentGuard for any content type.",
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
				MarkdownDescription: "Content plugin type. Either `certguard` or `core`.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"certguard",
						"core",
					),
				},
			},
			"plugin_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Plugin sub-type. One of `rhsm`, `x509`, `composite`, `content_redirect`, `header` or `rbac`",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"rhsm",
						"x509",
						"composite",
						"content_redirect",
						"header",
						"rbac",
					),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "A unique name for this ContentGuard.",
			},
			"description": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "A description for this ContentGuard.",
			},
			// "roles": schema.ListNestedAttribute{
			// 	Required:            false,
			// 	MarkdownDescription: "Roles that should be assigned to this ContentGuard.",
			// 	NestedObject: schema.NestedAttributeObject{
			// 		Attributes: map[string]schema.Attribute{
			// 			"users": schema.ListAttribute{
			// 				Required: false,
			// 			},
			// 			"groups": schema.ListAttribute{
			// 				Required: false,
			// 			},
			// 			"role": schema.StringAttribute{
			// 				Required: false,
			// 			},
			// 		},
			// 	},
			// },

			// Contentguards: Header exclusive
			"header_name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "`header_name` for Header. Supported only by Contentguards: Header.",
			},
			"header_value": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "`header_value` for Header. Supported only by Contentguards: Header.",
			},
			"jq_filter": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "`jq_filter` for Header. Supported only by Contentguards: Header.",
			},

			// Contentguards: X509 exclusive
			// Contentguards: Rhsm exclusive
			"ca_certificate": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "`ca_certificate` for X509 or Rhsm. Supported only by Contentguards: X509 or Rhsm",
			},

			// Contentguards: Composite exclusive
			"guards": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "`guards` for Composite. Supported only by Contentguards: Composite.",
			},

			// Contentguards: Rbac exclusive
			"users": schema.ListNestedAttribute{
				Optional: false,
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"username": schema.StringAttribute{
							Optional: true,
						},
						"pulp_href": schema.StringAttribute{
							Optional: true,
						},
						"prn": schema.StringAttribute{
							Optional: true,
						},
					},
				},
				MarkdownDescription: "`users` allowed to have role-based access. Supported only by Contentguards: Rbac.",
			},
			"groups": schema.ListNestedAttribute{
				Optional: false,
				Required: false,
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Optional: true,
						},
						"pulp_href": schema.StringAttribute{
							Optional: true,
						},
						"prn": schema.StringAttribute{
							Optional: true,
						},
						"id": schema.NumberAttribute{
							Optional: true,
						},
					},
				},
				MarkdownDescription: "`groups` allowed to have role-based access. Supported only by Contentguards: Rbac.",
			},
		},
	}
}

func (r *pulpContentGuardResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func buildContentGuardBody(ctx context.Context, plan PulpContentGuardModel) map[string]any {
	body := map[string]any{
		"name":        plan.Name.ValueString(),
		"description": plan.Description.ValueString(),
	}

	isCore := plan.ContentType.ValueString() == "core"
	isCertguard := plan.ContentType.ValueString() == "certguard"
	pluginName := plan.PluginName.ValueString()

	if isCore {
		switch pluginName {
		case "header":
			body["header_name"] = plan.HeaderName.ValueString()
			body["header_value"] = plan.HeaderValue.ValueString()
			body["jq_filter"] = plan.JqFilter.ValueString()
		case "composite":
			guards, _ := plan.Guards.ToListValue(ctx)
			body["guards"] = guards
		}
	} else if isCertguard {
		switch pluginName {
		case "x509", "rhsm":
			body["ca_certificate"] = plan.CaCertificate.ValueString()
		}
	}

	return body
}

func (r *pulpContentGuardResource) resourcePath(plan PulpContentGuardModel) string {
	return client.BuildResourcePath("contentguards", plan.ContentType.ValueString(), plan.PluginName.ValueString())
}

// Hydrate the model from a Pulp API response map.
func hydrateContentGuardModel(ctx context.Context, data map[string]any, model *PulpContentGuardModel) {
	if v, ok := data["pulp_href"].(string); ok {
		model.PulpHref = types.StringValue(v)
	}
	if v, ok := data["name"].(string); ok {
		model.Name = types.StringValue(v)
	}
	if v, ok := data["description"].(string); ok {
		model.Description = types.StringValue(v)
	}

	// Contentguards: Header exclusive
	if v, ok := data["header_name"].(string); ok {
		model.HeaderName = types.StringValue(v)
	}
	if v, ok := data["header_value"].(string); ok {
		model.HeaderValue = types.StringValue(v)
	}
	if v, ok := data["jq_filter"].(string); ok {
		model.JqFilter = types.StringValue(v)
	}

	// Contentguards: X509 / Rhsm exclusive
	if v, ok := data["ca_certificate"].(string); ok {
		model.CaCertificate = types.StringValue(v)
	}

	// Contentguards: Composite exclusive
	if v, ok := data["guards"].([]any); ok {
		guardElems := make([]types.String, 0, len(v))
		for _, g := range v {
			if s, ok := g.(string); ok {
				guardElems = append(guardElems, types.StringValue(s))
			}
		}
		guardList, diags := types.ListValueFrom(ctx, types.StringType, guardElems)
		if !diags.HasError() {
			model.Guards = guardList
		}
	}

	// Contentguards: Rbac exclusive — users (nested objects)
	userAttrTypes := map[string]attr.Type{
		"username":  types.StringType,
		"pulp_href": types.StringType,
		"prn":       types.StringType,
	}
	userObjType := types.ObjectType{AttrTypes: userAttrTypes}

	if v, ok := data["users"].([]any); ok {
		userVals := make([]attr.Value, 0, len(v))
		for _, u := range v {
			if userMap, ok := u.(map[string]any); ok {
				username := types.StringNull()
				if s, ok := userMap["username"].(string); ok {
					username = types.StringValue(s)
				}
				pulpHref := types.StringNull()
				if s, ok := userMap["pulp_href"].(string); ok {
					pulpHref = types.StringValue(s)
				}
				prn := types.StringNull()
				if s, ok := userMap["prn"].(string); ok {
					prn = types.StringValue(s)
				}

				obj, diags := types.ObjectValue(userAttrTypes, map[string]attr.Value{
					"username":  username,
					"pulp_href": pulpHref,
					"prn":       prn,
				})
				if !diags.HasError() {
					userVals = append(userVals, obj)
				}
			}
		}
		userList, diags := types.ListValue(userObjType, userVals)
		if !diags.HasError() {
			model.Users = userList
		}
	} else {
		model.Users, _ = types.ListValue(userObjType, []attr.Value{})
	}

	// Contentguards: Rbac exclusive — groups (nested objects)
	groupAttrTypes := map[string]attr.Type{
		"name":      types.StringType,
		"pulp_href": types.StringType,
		"prn":       types.StringType,
		"id":        types.NumberType,
	}
	groupObjType := types.ObjectType{AttrTypes: groupAttrTypes}

	if v, ok := data["groups"].([]any); ok {
		groupVals := make([]attr.Value, 0, len(v))
		for _, g := range v {
			if groupMap, ok := g.(map[string]any); ok {
				name := types.StringNull()
				if s, ok := groupMap["name"].(string); ok {
					name = types.StringValue(s)
				}
				pulpHref := types.StringNull()
				if s, ok := groupMap["pulp_href"].(string); ok {
					pulpHref = types.StringValue(s)
				}
				prn := types.StringNull()
				if s, ok := groupMap["prn"].(string); ok {
					prn = types.StringValue(s)
				}
				id := types.NumberNull()
				if n, ok := groupMap["id"].(float64); ok {
					id = types.NumberValue(big.NewFloat(n))
				}

				obj, diags := types.ObjectValue(groupAttrTypes, map[string]attr.Value{
					"name":      name,
					"pulp_href": pulpHref,
					"prn":       prn,
					"id":        id,
				})
				if !diags.HasError() {
					groupVals = append(groupVals, obj)
				}
			}
		}
		groupList, diags := types.ListValue(groupObjType, groupVals)
		if !diags.HasError() {
			model.Groups = groupList
		}
	} else {
		model.Groups, _ = types.ListValue(groupObjType, []attr.Value{})
	}
}

func (r *pulpContentGuardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PulpContentGuardModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := buildContentGuardBody(ctx, plan)
	resPath := r.resourcePath(plan)

	result, err := r.client.Create(ctx, resPath, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create ContentGuard", err.Error())
		return
	}

	hydrateContentGuardModel(ctx, result, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pulpContentGuardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PulpContentGuardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.ReadByHref(ctx, state.PulpHref.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read ContentGuard", err.Error())
		return
	}
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	hydrateContentGuardModel(ctx, result, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *pulpContentGuardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan PulpContentGuardModel
	var state PulpContentGuardModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := buildContentGuardBody(ctx, plan)

	result, err := r.client.Update(ctx, state.PulpHref.ValueString(), body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update ContentGuard", err.Error())
		return
	}

	// Preserve content_type and plugin_name from state (they force replace)
	plan.ContentType = state.ContentType
	plan.PluginName = state.PluginName

	hydrateContentGuardModel(ctx, result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pulpContentGuardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PulpContentGuardModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(ctx, state.PulpHref.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete ContentGuard", err.Error())
		return
	}
}

func (r *pulpContentGuardResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	pulpHref := req.ID

	// Example href: /pulp/api/v3/contentguards/core/content_redirect/<uuid>/
	// Parse content_type and plugin_name from the href
	parts := strings.Split(strings.Trim(pulpHref, "/"), "/")
	// parts: ["pulp", "api", "v3", "contentguards", "<content_type>", "<plugin_name>", "<uuid>"]
	if len(parts) < 7 {
		resp.Diagnostics.AddError("Invalid pulp_href", "Could not parse content_type and plugin_name from pulp_href")
		return
	}

	contentType := parts[4]
	pluginName := parts[5]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("pulp_href"), pulpHref)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("content_type"), contentType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("plugin_name"), pluginName)...)
}
