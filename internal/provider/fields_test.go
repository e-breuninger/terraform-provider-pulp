// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"math/big"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// declaredResources returns every resource the provider exposes, keyed by its
// terraform type name.
func declaredResources(t *testing.T) map[string]resource.Resource {
	t.Helper()

	out := map[string]resource.Resource{}
	for _, newResource := range (&PulpProvider{}).Resources(context.Background()) {
		r := newResource()
		var resp resource.MetadataResponse
		r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "pulp"}, &resp)
		out[resp.TypeName] = r
	}
	return out
}

// TestFieldTablesMatchModels is the guardrail that lets a field table be the
// single declaration of an attribute: it fails if a table entry has no model
// member to hydrate into, or a model member no table entry to describe it.
func TestFieldTablesMatchModels(t *testing.T) {
	for name, r := range declaredResources(t) {
		table, ok := r.(fieldTable)
		if !ok {
			continue
		}

		t.Run(name, func(t *testing.T) {
			declared := map[string]bool{}
			for _, f := range table.fieldTable() {
				if declared[f.Name] {
					t.Errorf("attribute %q is declared twice", f.Name)
				}
				declared[f.Name] = true
			}

			modelled := modelValues(table.newModel())

			for _, f := range table.fieldTable() {
				member, ok := modelled[f.Name]
				if !ok {
					t.Errorf("attribute %q has no `tfsdk:%q` member on the model", f.Name, f.Name)
					continue
				}
				// The generic body and hydration code asserts each member to
				// the type its Kind implies, so a mismatch would silently
				// drop the attribute.
				if got, want := member.Type(), f.goType(); got != want {
					t.Errorf("attribute %q is modelled as %s but its Kind needs %s", f.Name, got, want)
				}
			}
			for attr := range modelled {
				if !declared[attr] {
					t.Errorf("model member `tfsdk:%q` has no entry in the field table", attr)
				}
			}
		})
	}
}

// TestFieldTablesAreWellFormed checks the invariants the generic schema,
// body and hydration code relies on, so a malformed declaration fails here
// rather than at plan time inside Terraform.
func TestFieldTablesAreWellFormed(t *testing.T) {
	for name, r := range declaredResources(t) {
		table, ok := r.(fieldTable)
		if !ok {
			continue
		}
		features := featureSet(nil)
		if ft, ok := r.(featureTable); ok {
			features = ft.featureTable()
		}

		t.Run(name, func(t *testing.T) {
			for _, f := range table.fieldTable() {
				switch {
				case f.Required && f.Optional:
					t.Errorf("%q: Required and Optional are mutually exclusive", f.Name)
				case f.Required && f.Computed:
					t.Errorf("%q: Required and Computed are mutually exclusive", f.Name)
				case !f.Required && !f.Optional && !f.Computed:
					t.Errorf("%q: must be at least one of Required, Optional or Computed", f.Name)
				}

				if f.ReadOnly && !f.Computed {
					t.Errorf("%q: ReadOnly attributes must be Computed, Pulp assigns them", f.Name)
				}
				if f.Required && f.Nullable {
					t.Errorf("%q: a Required attribute is never null, drop Nullable", f.Name)
				}
				if f.Local && (f.ReadOnly || f.WriteOnly) {
					t.Errorf("%q: Local already implies neither reading nor writing", f.Name)
				}
				if f.Kind == fieldObjectList && len(f.Nested) == 0 {
					t.Errorf("%q: an object list needs Nested attributes", f.Name)
				}
				if (f.Kind == fieldNumber || f.Kind == fieldObjectList) && !f.ReadOnly && !f.Local {
					t.Errorf("%q: numbers and object lists are never written back, mark them ReadOnly or Local", f.Name)
				}

				if f.Feature == "" {
					continue
				}
				if features == nil {
					t.Errorf("%q: gated on feature %q but the resource has no variants", f.Name, f.Feature)
					continue
				}
				if features.variantsWith(f.Feature) == "" {
					t.Errorf("%q: gated on feature %q, which no variant supports", f.Name, f.Feature)
				}
			}
		})
	}
}

// TestSchemasBuild renders every resource schema, which is also the only way
// the panics in schemaAttribute can fire.
func TestSchemasBuild(t *testing.T) {
	for name, r := range declaredResources(t) {
		t.Run(name, func(t *testing.T) {
			var resp resource.SchemaResponse
			r.Schema(context.Background(), resource.SchemaRequest{}, &resp)

			if resp.Diagnostics.HasError() {
				t.Fatalf("schema diagnostics: %v", resp.Diagnostics)
			}
			if len(resp.Schema.Attributes) == 0 {
				t.Fatal("schema has no attributes")
			}
			if _, ok := resp.Schema.Attributes["pulp_href"]; !ok {
				t.Error("every Pulp resource is identified by pulp_href")
			}
		})
	}
}

func TestBuildBody(t *testing.T) {
	ctx := context.Background()
	fields := []field{
		{Name: "pulp_href", Kind: fieldString, Computed: true, ReadOnly: true},
		{Name: "content_type", Kind: fieldString, Required: true, Local: true},
		{Name: "name", Kind: fieldString, Required: true},
		{Name: "clearable", Kind: fieldString, Optional: true, Nullable: true},
		{Name: "keepable", Kind: fieldString, Optional: true},
		{Name: "gated", Kind: fieldString, Optional: true, Feature: "gated"},
	}

	type model struct {
		PulpHref    types.String `tfsdk:"pulp_href"`
		ContentType types.String `tfsdk:"content_type"`
		Name        types.String `tfsdk:"name"`
		Clearable   types.String `tfsdk:"clearable"`
		Keepable    types.String `tfsdk:"keepable"`
		Gated       types.String `tfsdk:"gated"`
	}

	plan := model{
		PulpHref:    types.StringValue("/pulp/api/v3/remotes/npm/npm/abc/"),
		ContentType: types.StringValue("npm"),
		Name:        types.StringValue("example"),
		Clearable:   types.StringNull(),
		Keepable:    types.StringNull(),
		Gated:       types.StringValue("set"),
	}

	t.Run("feature supported", func(t *testing.T) {
		body := buildBody(ctx, fields, &plan, func(string) bool { return true })
		want := map[string]any{"name": "example", "clearable": nil, "gated": "set"}
		if !reflect.DeepEqual(body, want) {
			t.Errorf("got %#v, want %#v", body, want)
		}
	})

	t.Run("feature unsupported", func(t *testing.T) {
		body := buildBody(ctx, fields, &plan, func(string) bool { return false })
		if _, ok := body["gated"]; ok {
			t.Error("an unsupported attribute must not reach the request body")
		}
	})

	t.Run("unknown is omitted", func(t *testing.T) {
		unknown := plan
		unknown.Keepable = types.StringUnknown()
		unknown.Clearable = types.StringUnknown()
		body := buildBody(ctx, fields, &unknown, func(string) bool { return true })
		for _, key := range []string{"keepable", "clearable"} {
			if _, ok := body[key]; ok {
				t.Errorf("unknown attribute %q must be omitted, not sent as null", key)
			}
		}
	})
}

func TestHydrateModel(t *testing.T) {
	ctx := context.Background()
	fields := []field{
		{Name: "pulp_href", Kind: fieldString, Computed: true, ReadOnly: true},
		{Name: "content_type", Kind: fieldString, Required: true, Local: true},
		{Name: "name", Kind: fieldString, Required: true},
		{Name: "remote", Kind: fieldString, Optional: true, EmptyIsNull: true},
		{Name: "password", Kind: fieldString, Optional: true, WriteOnly: true},
		{Name: "enabled", Kind: fieldBool, Optional: true},
	}

	type model struct {
		PulpHref    types.String `tfsdk:"pulp_href"`
		ContentType types.String `tfsdk:"content_type"`
		Name        types.String `tfsdk:"name"`
		Remote      types.String `tfsdk:"remote"`
		Password    types.String `tfsdk:"password"`
		Enabled     types.Bool   `tfsdk:"enabled"`
	}

	m := model{
		ContentType: types.StringValue("npm"),
		Password:    types.StringValue("secret"),
	}
	hydrateModel(ctx, fields, map[string]any{
		"pulp_href": "/pulp/api/v3/remotes/npm/npm/abc/",
		"name":      "example",
		"remote":    "",
		"enabled":   true,
	}, &m)

	if got := m.PulpHref.ValueString(); got != "/pulp/api/v3/remotes/npm/npm/abc/" {
		t.Errorf("pulp_href = %q", got)
	}
	if !m.Remote.IsNull() {
		t.Errorf(`remote = %v, Pulp reports an unset href as "" and it must read back as null`, m.Remote)
	}
	if got := m.Password.ValueString(); got != "secret" {
		t.Errorf("password = %q, a write-only attribute must keep its configured value", got)
	}
	if got := m.ContentType.ValueString(); got != "npm" {
		t.Errorf("content_type = %q, a local attribute must not be hydrated", got)
	}
	if !m.Enabled.ValueBool() {
		t.Error("enabled did not hydrate")
	}
}

func TestFeatureSetAccessors(t *testing.T) {
	f := featureSet{
		"container/container":    {featurePrivate: true},
		"container/pull-through": {featurePrivate: true, featureRemote: true},
		"npm/npm":                {featureRemote: true},
	}

	if !f.knows("npm", "npm") {
		t.Error("npm/npm should be known")
	}
	if f.knows("npm", "pull-through") {
		t.Error("a combination of two valid halves is not itself valid")
	}
	if !f.supports("container", "pull-through", featureRemote) {
		t.Error("container/pull-through supports remote")
	}
	if f.supports("container", "container", featureRemote) {
		t.Error("container/container does not support remote")
	}
	if got, want := f.contentTypes(), []string{"container", "npm"}; !slices.Equal(got, want) {
		t.Errorf("contentTypes() = %v, want %v", got, want)
	}
	if got, want := f.pluginNames(), []string{"container", "npm", "pull-through"}; !slices.Equal(got, want) {
		t.Errorf("pluginNames() = %v, want %v", got, want)
	}
	if got, want := f.variantsWith(featureRemote), "`container/pull-through`, `npm/npm`"; got != want {
		t.Errorf("variantsWith() = %q, want %q", got, want)
	}
}

// TestVariantValidatorsCoverEveryVariant checks that the OneOf validators
// derived from a featureSet accept every combination it declares.
func TestVariantValidatorsCoverEveryVariant(t *testing.T) {
	for name, r := range declaredResources(t) {
		ft, ok := r.(featureTable)
		if !ok || ft.featureTable() == nil {
			continue
		}

		t.Run(name, func(t *testing.T) {
			features := ft.featureTable()
			contentTypes := features.contentTypes()
			pluginNames := features.pluginNames()

			for _, variant := range features.variants() {
				contentType, pluginName, ok := splitVariant(variant)
				if !ok {
					t.Fatalf("malformed variant key %q", variant)
				}
				if !slices.Contains(contentTypes, contentType) {
					t.Errorf("%s: content_type %q missing from the validator", variant, contentType)
				}
				if !slices.Contains(pluginNames, pluginName) {
					t.Errorf("%s: plugin_name %q missing from the validator", variant, pluginName)
				}
			}
		})
	}
}

func splitVariant(variant string) (contentType, pluginName string, ok bool) {
	contentType, pluginName, ok = strings.Cut(variant, "/")
	return contentType, pluginName, ok
}

func TestUserIDPath(t *testing.T) {
	for _, tc := range []struct {
		in   types.Number
		want string
	}{
		{types.NumberValue(bigFloat(3)), "3"},
		{types.NumberValue(bigFloat(1234567)), "1234567"},
		{types.NumberNull(), ""},
	} {
		if got := userIDPath(tc.in); got != tc.want {
			t.Errorf("userIDPath(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func bigFloat(v float64) *big.Float { return big.NewFloat(v) }
