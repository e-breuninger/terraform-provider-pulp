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

// pulpResource is the CRUD lifecycle every Pulp resource shares: POST to a
// collection, then GET/PATCH/DELETE the pulp_href the server returns. A
// resource embeds it and supplies only its declaration. M is the resource's
// model, whose `tfsdk` tags match the Name of each field.
//
// See README.md.
type pulpResource[M any] struct {
	client *client.PulpClient

	typeName    string // terraform type suffix: "distribution" -> pulp_distribution
	label       string // name used in diagnostics, e.g. "Distribution"
	description string
	collection  string // Pulp API collection segment, e.g. "distributions"
	// features is nil for resources served at a single endpoint.
	features featureSet
	fields   []field

	// resourcePath overrides the collection a create POSTs to, for resources
	// nested under another one.
	resourcePath func(model *M) string
	// afterHydrate derives values that are not read straight from the
	// response.
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

// configureClient unwraps the provider data a resource receives.
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

// variant reads the pair that selects the endpoint, empty for resources
// without variants.
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

// body renders the plan into a request body, gating attributes on the
// featureSet.
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

	// content_type and plugin_name require replacement, so the plan's variant
	// always matches the existing href.
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

// ImportState imports by pulp_href, recovering content_type and plugin_name
// from it: /pulp/api/v3/remotes/<content_type>/<plugin_name>/<id>/.
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

// ValidateConfig rejects at plan time what Pulp could only answer with a 404
// or a validation error.
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
		// An unresolved reference at plan time; Pulp has the last word.
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

// validateFeatures reports attributes set in the config but not accepted by
// the chosen variant, which would otherwise be dropped from the body and read
// back as null.
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

// fieldTable and featureTable expose a resource's declaration to the tests
// that check it.
type fieldTable interface {
	fieldTable() []field
	newModel() any
}

func (r *pulpResource[M]) fieldTable() []field { return r.fields }

func (r *pulpResource[M]) newModel() any {
	var m M
	return &m
}

type featureTable interface {
	featureTable() featureSet
	collectionName() string
}

func (r *pulpResource[M]) featureTable() featureSet { return r.features }

func (r *pulpResource[M]) collectionName() string { return r.collection }
