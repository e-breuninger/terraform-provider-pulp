// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"github.com/e-breuninger/terraform-provider-pulp/internal"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/numberplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// A field is the single declaration of one attribute of a Pulp resource. The
// terraform schema, the JSON request body and the hydration of the model from
// a Pulp response are all derived from it, so adding an attribute to a
// resource means adding one entry to that resource's field table plus the
// matching struct member on its model — and nothing else.
//
// TestFieldTablesMatchModels keeps the two halves in step: it fails if a
// table entry has no `tfsdk` struct member, or a struct member has no entry.
type field struct {
	// Name is the Pulp field name. It doubles as the terraform attribute
	// name, the request body key and the response key — Pulp is consistent
	// about this, and relying on it is what lets one entry drive all three.
	Name string
	Kind fieldKind
	// Description is rendered into the docs as the attribute's
	// MarkdownDescription.
	Description string

	Required  bool
	Optional  bool
	Computed  bool
	Sensitive bool

	// RequiresReplace forces replacement when the value changes. Used for the
	// attributes that select which Pulp endpoint the resource lives on: they
	// cannot be PATCHed, the resource has to be recreated elsewhere.
	RequiresReplace bool
	// UseStateForUnknown keeps a Computed attribute at its prior value during
	// planning instead of showing "(known after apply)" on every change.
	UseStateForUnknown bool

	// ReadOnly marks a value Pulp computes and refuses in a request body.
	ReadOnly bool
	// WriteOnly marks a value Pulp accepts but never returns — credentials,
	// mostly. Hydration skips these, because overwriting the configured value
	// with the null Pulp reports would fail the apply with "Provider produced
	// inconsistent result after apply".
	WriteOnly bool
	// Nullable marks a Pulp field that accepts an explicit JSON null, so that
	// removing the attribute from the config clears it server-side. Without
	// it a null is omitted from the body instead, because Pulp rejects nulls
	// on its many non-nullable fields with "This field may not be null."
	Nullable bool
	// EmptyIsNull marks a Pulp field reported as "" when unset, which has to
	// be read back as null so it round-trips against an absent attribute.
	EmptyIsNull bool
	// Local marks an attribute that is not a Pulp field at all. content_type
	// and plugin_name pick which endpoint the resource lives on rather than
	// being stored on it, so they are neither sent in a request body nor read
	// back from a response.
	Local bool

	// Feature gates the attribute on the resource's featureSet: it is only
	// written to a request body, and only accepted in a config, for the
	// content_type/plugin_name variants that list it. Empty means every
	// variant supports the attribute.
	Feature string

	// Nested describes the element attributes of a fieldObjectList.
	Nested []field

	StringValidators []validator.String
	ListValidators   []validator.List
}

type fieldKind int

const (
	fieldString fieldKind = iota
	fieldBool
	fieldNumber
	fieldStringList
	fieldStringSet
	fieldLabels
	fieldObjectList
)

// attrType is the terraform type of a field, used to build nested object
// types for fieldObjectList.
func (f field) attrType() attr.Type {
	switch f.Kind {
	case fieldString:
		return types.StringType
	case fieldBool:
		return types.BoolType
	case fieldNumber:
		return types.NumberType
	case fieldStringList:
		return types.ListType{ElemType: types.StringType}
	case fieldStringSet:
		return types.SetType{ElemType: types.StringType}
	case fieldLabels:
		return types.MapType{ElemType: types.StringType}
	case fieldObjectList:
		return types.ListType{ElemType: types.ObjectType{AttrTypes: nestedAttrTypes(f.Nested)}}
	}
	panic(fmt.Sprintf("unhandled field kind %d for %q", f.Kind, f.Name))
}

func nestedAttrTypes(fs []field) map[string]attr.Type {
	out := make(map[string]attr.Type, len(fs))
	for _, f := range fs {
		out[f.Name] = f.attrType()
	}
	return out
}

// description is the attribute's rendered documentation. For a gated
// attribute it names the variants that accept it, so the generated docs say
// where it applies without anyone having to keep a second list in step.
func (f field) description(features featureSet) string {
	if f.Feature == "" || features == nil {
		return f.Description
	}
	return f.Description + " Only supported by: " + features.variantsWith(f.Feature) + "."
}

// schemaAttribute renders the field as a terraform-plugin-framework schema
// attribute. features is the resource's variant table, or nil for resources
// served at a single endpoint.
func (f field) schemaAttribute(features featureSet) schema.Attribute {
	description := f.description(features)

	switch f.Kind {
	case fieldString:
		return schema.StringAttribute{
			Required: f.Required, Optional: f.Optional, Computed: f.Computed,
			Sensitive:           f.Sensitive,
			MarkdownDescription: description,
			Validators:          f.StringValidators,
			PlanModifiers:       f.stringPlanModifiers(),
		}
	case fieldBool:
		return schema.BoolAttribute{
			Required: f.Required, Optional: f.Optional, Computed: f.Computed,
			MarkdownDescription: description,
			PlanModifiers:       f.boolPlanModifiers(),
		}
	case fieldNumber:
		return schema.NumberAttribute{
			Required: f.Required, Optional: f.Optional, Computed: f.Computed,
			MarkdownDescription: description,
			PlanModifiers:       f.numberPlanModifiers(),
		}
	case fieldStringList:
		return schema.ListAttribute{
			Required: f.Required, Optional: f.Optional, Computed: f.Computed,
			ElementType:         types.StringType,
			MarkdownDescription: description,
			Validators:          f.ListValidators,
			PlanModifiers:       f.listPlanModifiers(),
		}
	case fieldStringSet:
		return schema.SetAttribute{
			Required: f.Required, Optional: f.Optional, Computed: f.Computed,
			ElementType:         types.StringType,
			MarkdownDescription: description,
			PlanModifiers:       f.setPlanModifiers(),
		}
	case fieldLabels:
		return schema.MapAttribute{
			Required: f.Required, Optional: f.Optional, Computed: f.Computed,
			ElementType:         types.StringType,
			MarkdownDescription: description,
		}
	case fieldObjectList:
		return schema.ListNestedAttribute{
			Required: f.Required, Optional: f.Optional, Computed: f.Computed,
			MarkdownDescription: description,
			NestedObject:        schema.NestedAttributeObject{Attributes: fieldsSchema(f.Nested, nil)},
		}
	}
	panic(fmt.Sprintf("unhandled field kind %d for %q", f.Kind, f.Name))
}

func (f field) stringPlanModifiers() []planmodifier.String {
	var out []planmodifier.String
	if f.RequiresReplace {
		out = append(out, stringplanmodifier.RequiresReplace())
	}
	if f.UseStateForUnknown {
		out = append(out, stringplanmodifier.UseStateForUnknown())
	}
	return out
}

func (f field) boolPlanModifiers() []planmodifier.Bool {
	var out []planmodifier.Bool
	if f.RequiresReplace {
		out = append(out, boolplanmodifier.RequiresReplace())
	}
	if f.UseStateForUnknown {
		out = append(out, boolplanmodifier.UseStateForUnknown())
	}
	return out
}

func (f field) numberPlanModifiers() []planmodifier.Number {
	var out []planmodifier.Number
	if f.RequiresReplace {
		out = append(out, numberplanmodifier.RequiresReplace())
	}
	if f.UseStateForUnknown {
		out = append(out, numberplanmodifier.UseStateForUnknown())
	}
	return out
}

func (f field) listPlanModifiers() []planmodifier.List {
	var out []planmodifier.List
	if f.RequiresReplace {
		out = append(out, listplanmodifier.RequiresReplace())
	}
	if f.UseStateForUnknown {
		out = append(out, listplanmodifier.UseStateForUnknown())
	}
	return out
}

func (f field) setPlanModifiers() []planmodifier.Set {
	var out []planmodifier.Set
	if f.RequiresReplace {
		out = append(out, setplanmodifier.RequiresReplace())
	}
	if f.UseStateForUnknown {
		out = append(out, setplanmodifier.UseStateForUnknown())
	}
	return out
}

// fieldsSchema renders a whole field table as a schema attribute map.
func fieldsSchema(fs []field, features featureSet) map[string]schema.Attribute {
	attrs := make(map[string]schema.Attribute, len(fs))
	for _, f := range fs {
		attrs[f.Name] = f.schemaAttribute(features)
	}
	return attrs
}

// modelValues indexes a terraform model struct by its `tfsdk` tags, so the
// field table can address model members by their Pulp field name. model must
// be a pointer to a struct.
func modelValues(model any) map[string]reflect.Value {
	v := reflect.ValueOf(model)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()
	out := make(map[string]reflect.Value, t.NumField())
	for i := range t.NumField() {
		if tag, ok := t.Field(i).Tag.Lookup("tfsdk"); ok {
			out[tag] = v.Field(i)
		}
	}
	return out
}

// buildBody renders the writable fields of a plan into a Pulp request body.
//
// supports reports whether the variant being written accepts a gated
// attribute; pass nil for resources that have no content_type/plugin_name
// variants, in which case every gated attribute is skipped.
func buildBody(ctx context.Context, fs []field, plan any, supports func(feature string) bool) map[string]any {
	body := make(map[string]any, len(fs))
	values := modelValues(plan)

	for _, f := range fs {
		if f.ReadOnly || f.Local {
			continue
		}
		if f.Feature != "" && (supports == nil || !supports(f.Feature)) {
			continue
		}
		v, ok := values[f.Name]
		if !ok {
			continue
		}
		// The type assertions cannot fail: TestFieldTablesMatchModels
		// checks every model member against the Kind its field declares.
		switch f.Kind {
		case fieldString:
			if value, ok := valueOf[types.String](v); ok {
				internal.SetStr(body, f.Name, value, f.Nullable)
			}
		case fieldBool:
			if value, ok := valueOf[types.Bool](v); ok {
				internal.SetBool(body, f.Name, value, f.Nullable)
			}
		case fieldStringList:
			if value, ok := valueOf[types.List](v); ok {
				internal.SetStringList(ctx, body, f.Name, value)
			}
		case fieldStringSet:
			if value, ok := valueOf[types.Set](v); ok {
				internal.SetStringSet(ctx, body, f.Name, value)
			}
		case fieldLabels:
			if value, ok := valueOf[types.Map](v); ok {
				internal.SetLabels(ctx, body, value)
			}
		case fieldNumber, fieldObjectList:
			// Pulp assigns these (identifiers, computed role memberships) and
			// rejects them in a request body, so they are always ReadOnly or
			// Local and never reach this point.
		}
	}
	return body
}

// hydrateModel copies a Pulp API response into a terraform model, following
// the field table. Fields absent from the response become null, which is what
// Pulp means by omitting them.
func hydrateModel(ctx context.Context, fs []field, data map[string]any, model any) {
	values := modelValues(model)

	for _, f := range fs {
		if f.WriteOnly || f.Local {
			continue
		}
		v, ok := values[f.Name]
		if !ok {
			continue
		}
		switch f.Kind {
		case fieldString:
			if f.EmptyIsNull {
				v.Set(reflect.ValueOf(internal.StrOrNullNonEmpty(data, f.Name)))
			} else {
				v.Set(reflect.ValueOf(internal.StrOrNull(data, f.Name)))
			}
		case fieldBool:
			v.Set(reflect.ValueOf(internal.BoolOrNull(data, f.Name)))
		case fieldNumber:
			v.Set(reflect.ValueOf(internal.NumberOrNull(data, f.Name)))
		case fieldStringList:
			v.Set(reflect.ValueOf(internal.StringList(ctx, data, f.Name)))
		case fieldStringSet:
			v.Set(reflect.ValueOf(internal.StringSet(ctx, data, f.Name)))
		case fieldLabels:
			v.Set(reflect.ValueOf(internal.LabelsOrNull(ctx, data)))
		case fieldObjectList:
			v.Set(reflect.ValueOf(objectList(data, f)))
		}
	}
}

// objectList converts a list of Pulp objects (e.g. the users and groups a
// content guard grants access to) into a types.List of terraform objects,
// following the nested field table.
func objectList(data map[string]any, f field) types.List {
	attrTypes := nestedAttrTypes(f.Nested)
	objType := types.ObjectType{AttrTypes: attrTypes}

	raw, ok := data[f.Name].([]any)
	if !ok {
		empty, _ := types.ListValue(objType, []attr.Value{})
		return empty
	}

	elems := make([]attr.Value, 0, len(raw))
	for _, item := range raw {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		attrs := make(map[string]attr.Value, len(f.Nested))
		for _, nf := range f.Nested {
			switch nf.Kind {
			case fieldString:
				attrs[nf.Name] = internal.StrOrNull(m, nf.Name)
			case fieldBool:
				attrs[nf.Name] = internal.BoolOrNull(m, nf.Name)
			case fieldNumber:
				attrs[nf.Name] = internal.NumberOrNull(m, nf.Name)
			default:
				attrs[nf.Name] = types.StringNull()
			}
		}
		obj, diags := types.ObjectValue(attrTypes, attrs)
		if !diags.HasError() {
			elems = append(elems, obj)
		}
	}

	list, diags := types.ListValue(objType, elems)
	if diags.HasError() {
		return types.ListNull(objType)
	}
	return list
}

// valueOf reads an indexed model member as a concrete terraform value type.
func valueOf[T attr.Value](v reflect.Value) (T, bool) {
	value, ok := v.Interface().(T)
	return value, ok
}

// goType is the terraform value type a field of this kind must be modelled
// as. TestFieldTablesMatchModels checks each model member against it.
func (f field) goType() reflect.Type {
	switch f.Kind {
	case fieldString:
		return reflect.TypeFor[types.String]()
	case fieldBool:
		return reflect.TypeFor[types.Bool]()
	case fieldNumber:
		return reflect.TypeFor[types.Number]()
	case fieldStringList, fieldObjectList:
		return reflect.TypeFor[types.List]()
	case fieldStringSet:
		return reflect.TypeFor[types.Set]()
	case fieldLabels:
		return reflect.TypeFor[types.Map]()
	}
	panic(fmt.Sprintf("unhandled field kind %d for %q", f.Kind, f.Name))
}

// knownString reads a string attribute out of an indexed model, reporting
// false when the attribute is absent, null, or still unknown at plan time.
func knownString(values map[string]reflect.Value, name string) (string, bool) {
	v, ok := values[name]
	if !ok {
		return "", false
	}
	s, ok := valueOf[types.String](v)
	if !ok || s.IsNull() || s.IsUnknown() {
		return "", false
	}
	return s.ValueString(), true
}

// isNullOrUnknown reports whether an indexed model value carries no
// practitioner-supplied value.
func isNullOrUnknown(v reflect.Value) bool {
	av, ok := v.Interface().(attr.Value)
	if !ok {
		return true
	}
	return av.IsNull() || av.IsUnknown()
}

// hrefField is the computed pulp_href that identifies every Pulp resource.
func hrefField() field {
	return field{
		Name: "pulp_href", Kind: fieldString,
		Computed: true, ReadOnly: true, UseStateForUnknown: true,
		Description: "The `pulp_href` (used as the resource identifier).",
	}
}

// labelsField is the pulp_labels map shared by remotes, repositories and
// distributions.
func labelsField() field {
	return field{
		Name: "pulp_labels", Kind: fieldLabels,
		Optional: true, Computed: true,
		Description: "Key/value labels.",
	}
}

// variantFields are the two attributes that select which Pulp endpoint a
// resource lives on. Their accepted values, and the combinations listed in
// their documentation, come straight from the resource's featureSet, so they
// cannot drift from the variants the provider actually supports.
//
// Both require replacement: Pulp stores the resource under
// /<collection>/<content_type>/<plugin_name>/, so changing either means a
// different object at a different href, not an update of this one.
func variantFields(f featureSet) []field {
	return []field{
		{
			Name: "content_type", Kind: fieldString,
			Required: true, RequiresReplace: true, Local: true,
			Description: "Pulp content plugin type. Together with `plugin_name` it selects the API endpoint. Supported combinations: " +
				f.variantsMarkdown() + ".",
			StringValidators: []validator.String{stringvalidator.OneOf(f.contentTypes()...)},
		},
		{
			Name: "plugin_name", Kind: fieldString,
			Required: true, RequiresReplace: true, Local: true,
			Description:      "Pulp plugin sub-type. See `content_type` for the supported combinations.",
			StringValidators: []validator.String{stringvalidator.OneOf(f.pluginNames()...)},
		},
	}
}

// variantResourceFields prefixes a resource's own attributes with the
// pulp_href and the content_type/plugin_name pair every variant-served
// resource has.
func variantResourceFields(f featureSet, own ...field) []field {
	return slices.Concat([]field{hrefField()}, variantFields(f), own)
}
