// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"reflect"

	"github.com/e-breuninger/terraform-provider-pulp/internal"
	"github.com/e-breuninger/terraform-provider-pulp/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// pulpResource implements the CRUD lifecycle every Pulp resource shares:
// POST to a collection to create, then GET/PATCH/DELETE the pulp_href the
// server hands back. A concrete resource embeds it and supplies only its
// declaration — the field table, the variants it supports, and its names.
//
// M is the resource's terraform model struct, whose `tfsdk` tags line up with
// the Name of each entry in fields.
type pulpResource[M any] struct {
	client *client.PulpClient

	// typeName is the terraform type suffix: "distribution" makes
	// pulp_distribution.
	typeName string
	// label names the resource in diagnostics, e.g. "Distribution".
	label string
	// description is the resource's MarkdownDescription.
	description string
	// collection is the Pulp API collection segment, e.g. "distributions".
	collection string
	// features declares the content_type/plugin_name variants this resource
	// is served at, and which attributes each one accepts. It is nil for the
	// resources that live at a single endpoint, such as groups and users.
	features featureSet
	// fields declares every attribute of the resource.
	fields []field

	// resourcePath overrides the collection path a create POSTs to. Only
	// needed by resources nested under another one, such as user roles.
	resourcePath func(model *M) string
	// afterHydrate runs after the generic hydration, for the few values that
	// have to be derived rather than read straight out of the response.
	afterHydrate func(ctx context.Context, data map[string]any, model *M)
}

var (
	_ resource.Resource                   = &pulpResource[struct{}]{}
	_ resource.ResourceWithConfigure      = &pulpResource[struct{}]{}
	_ resource.ResourceWithImportState    = &pulpResource[struct{}]{}
	_ resource.ResourceWithValidateConfig = &pulpResource[struct{}]{}
)

func (r *pulpResource[M]) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.typeName
}

func (r *pulpResource[M]) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: r.description,
		Attributes:          fieldsSchema(r.fields, r.features),
	}
}

func (r *pulpResource[M]) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req, resp)
}

// configureClient unwraps the provider data every resource and data source
// receives.
func configureClient(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *client.PulpClient {
	if req.ProviderData == nil {
		return nil
	}
	c, ok := req.ProviderData.(*client.PulpClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type",
			fmt.Sprintf("Expected *client.PulpClient, got %T", req.ProviderData))
		return nil
	}
	return c
}

// variant reads the content_type/plugin_name pair that selects this
// resource's Pulp endpoint. It returns empty strings for resources that have
// no variants.
func (r *pulpResource[M]) variant(model *M) (contentType, pluginName string) {
	values := modelValues(model)
	contentType, _ = knownString(values, "content_type")
	pluginName, _ = knownString(values, "plugin_name")
	return contentType, pluginName
}

// path is the collection a create POSTs to.
func (r *pulpResource[M]) path(model *M) string {
	if r.resourcePath != nil {
		return r.resourcePath(model)
	}
	if r.features == nil {
		return r.collection
	}
	contentType, pluginName := r.variant(model)
	return client.BuildResourcePath(r.collection, contentType, pluginName)
}

// body renders the plan into a Pulp request body, gating variant-specific
// attributes on the resource's featureSet.
func (r *pulpResource[M]) body(ctx context.Context, model *M) map[string]any {
	if r.features == nil {
		return buildBody(ctx, r.fields, model, nil)
	}
	contentType, pluginName := r.variant(model)
	return buildBody(ctx, r.fields, model, func(feature string) bool {
		return r.features.supports(contentType, pluginName, feature)
	})
}

func (r *pulpResource[M]) hydrate(ctx context.Context, data map[string]any, model *M) {
	tflog.Debug(ctx, "Hydrating "+r.typeName+" model", map[string]any{
		"data": fmt.Sprintf("%+v", data),
	})
	hydrateModel(ctx, r.fields, data, model)
	if r.afterHydrate != nil {
		r.afterHydrate(ctx, data, model)
	}
}

// href reads the pulp_href that identifies an existing resource.
func (r *pulpResource[M]) href(model *M) string {
	href, _ := knownString(modelValues(model), "pulp_href")
	return href
}

func (r *pulpResource[M]) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan M
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Create(ctx, r.path(&plan), r.body(ctx, &plan))
	if err != nil {
		resp.Diagnostics.AddError("Failed to create "+r.label, err.Error())
		return
	}

	r.hydrate(ctx, result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pulpResource[M]) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state M
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.ReadByHref(ctx, r.href(&state))
	if err != nil {
		resp.Diagnostics.AddError("Failed to read "+r.label, err.Error())
		return
	}
	if result == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.hydrate(ctx, result, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *pulpResource[M]) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state M
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// content_type and plugin_name require replacement, so the variant in the
	// plan always matches the one the existing href lives on.
	result, err := r.client.Update(ctx, r.href(&state), r.body(ctx, &plan))
	if err != nil {
		resp.Diagnostics.AddError("Failed to update "+r.label, err.Error())
		return
	}

	r.hydrate(ctx, result, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pulpResource[M]) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state M
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(ctx, r.href(&state)); err != nil {
		resp.Diagnostics.AddError("Failed to delete "+r.label, err.Error())
	}
}

// ImportState imports by pulp_href. For resources served per variant the
// content_type and plugin_name are recovered from the href itself, since the
// href encodes them: /pulp/api/v3/remotes/<content_type>/<plugin_name>/<id>/.
func (r *pulpResource[M]) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("pulp_href"), req.ID)...)
	if r.features == nil {
		return
	}

	contentType, pluginName, err := internal.ParseHrefVariant(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid pulp_href", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("content_type"), contentType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("plugin_name"), pluginName)...)
}

// ValidateConfig rejects configurations Pulp could only answer with a 404 or
// a validation error, and does it at plan time with a message that says what
// is actually supported.
func (r *pulpResource[M]) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	if r.features == nil {
		return
	}

	var config M
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	values := modelValues(&config)
	contentType, okCT := knownString(values, "content_type")
	pluginName, okPN := knownString(values, "plugin_name")
	if !okCT || !okPN {
		// Either half can be an unresolved reference at plan time; Pulp will
		// have the last word.
		return
	}

	if !r.features.knows(contentType, pluginName) {
		resp.Diagnostics.AddAttributeError(path.Root("plugin_name"),
			"Unsupported content_type/plugin_name combination",
			fmt.Sprintf("Pulp does not serve %s of type %q. Supported combinations are: %s.",
				r.collection, variantKey(contentType, pluginName), r.features.variantsMarkdown()))
		return
	}

	r.validateFeatures(values, contentType, pluginName, &resp.Diagnostics)
}

// validateFeatures reports attributes that are set in the config but not
// accepted by the chosen variant. Without this the attribute would be
// silently dropped from the request body and then read back as null, which
// surfaces much later as a confusing "inconsistent result after apply".
func (r *pulpResource[M]) validateFeatures(values map[string]reflect.Value, contentType, pluginName string, diags *diag.Diagnostics) {
	for _, f := range r.fields {
		if f.Feature == "" || f.ReadOnly || r.features.supports(contentType, pluginName, f.Feature) {
			continue
		}
		v, ok := values[f.Name]
		if !ok || isNullOrUnknown(v) {
			continue
		}
		diags.AddAttributeError(path.Root(f.Name),
			fmt.Sprintf("Attribute %q is not supported by this variant", f.Name),
			fmt.Sprintf("A %s of type %q does not accept %q. It is accepted by: %s.",
				r.label, variantKey(contentType, pluginName), f.Name, r.features.variantsWith(f.Feature)))
	}
}

// fieldTable exposes a resource's declaration for tests and tooling. Every
// resource built on pulpResource implements it.
type fieldTable interface {
	fieldTable() []field
	newModel() any
}

func (r *pulpResource[M]) fieldTable() []field { return r.fields }

func (r *pulpResource[M]) newModel() any {
	var m M
	return &m
}

// featureTable exposes the variants a resource is served at, or nil for the
// resources that live at a single endpoint.
type featureTable interface {
	featureTable() featureSet
	collectionName() string
}

func (r *pulpResource[M]) featureTable() featureSet { return r.features }

func (r *pulpResource[M]) collectionName() string { return r.collection }
