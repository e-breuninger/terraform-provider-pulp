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

// A field declares one attribute of a resource. Name is the Pulp field name
// and doubles as the terraform attribute, the body key and the response key.
type field struct {
	Name        string
	Kind        fieldKind
	Description string

	Required  bool
	Optional  bool
	Computed  bool
	Sensitive bool

	RequiresReplace    bool
	UseStateForUnknown bool

	ReadOnly  bool // Pulp assigns it and rejects it in a body.
	WriteOnly bool // Pulp accepts it but never returns it, so never hydrate it.
	Nullable  bool // Pulp accepts an explicit null, so clearing the config clears it.
	// EmptyIsNull reads Pulp's "" for an unset field back as null.
	EmptyIsNull bool
	// Local is not a Pulp field: content_type and plugin_name pick the
	// endpoint rather than being stored on the resource.
	Local bool

	// Feature limits the attribute to the variants of the resource's
	// featureSet that list it. Empty means every variant accepts it.
	Feature string

	// Nested holds the element attributes of a fieldObjectList.
	Nested []field

	StringValidators []validator.String
	NumberValidators []validator.Number
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

// description names the accepted variants for a gated attribute, so the docs
// cannot drift from the featureSet.
func (f field) description(features featureSet) string {
	if f.Feature == "" || features == nil {
		return f.Description
	}
	return f.Description + " Only supported by: " + features.variantsWith(f.Feature) + "."
}

// schemaAttribute renders the field as a schema attribute. features is nil
// for resources served at a single endpoint.
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
			Validators:          f.NumberValidators,
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

// fieldsSchema renders a field table as a schema attribute map.
func fieldsSchema(fs []field, features featureSet) map[string]schema.Attribute {
	attrs := make(map[string]schema.Attribute, len(fs))
	for _, f := range fs {
		attrs[f.Name] = f.schemaAttribute(features)
	}
	return attrs
}

// modelValues indexes a model struct by its `tfsdk` tags. model must be a
// pointer to a struct.
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
// supports reports whether the variant accepts a gated attribute; nil skips
// every gated attribute.
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
		// TestFieldTablesMatchModels checks every member against its Kind,
		// so these assertions cannot fail.
		switch f.Kind {
		case fieldString:
			if value, ok := valueOf[types.String](v); ok {
				internal.SetStr(body, f.Name, value, f.Nullable)
			}
		case fieldBool:
			if value, ok := valueOf[types.Bool](v); ok {
				internal.SetBool(body, f.Name, value, f.Nullable)
			}
		case fieldNumber:
			if value, ok := valueOf[types.Number](v); ok {
				internal.SetNumber(body, f.Name, value, f.Nullable)
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
		case fieldObjectList:
			// Always ReadOnly, so never reached.
		}
	}
	return body
}

// hydrateModel copies a Pulp response into a model. Absent fields become
// null.
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

// objectList converts a list of Pulp objects into a types.List, following the
// nested field table.
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

func valueOf[T attr.Value](v reflect.Value) (T, bool) {
	value, ok := v.Interface().(T)
	return value, ok
}

// goType is the value type a field of this kind must be modelled as.
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

// knownString reads a string member, reporting false when it is absent, null
// or unknown.
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

// isNullOrUnknown reports whether a member carries no configured value.
func isNullOrUnknown(v reflect.Value) bool {
	av, ok := v.Interface().(attr.Value)
	if !ok {
		return true
	}
	return av.IsNull() || av.IsUnknown()
}

// hrefField identifies every Pulp resource.
func hrefField() field {
	return field{
		Name: "pulp_href", Kind: fieldString,
		Computed: true, ReadOnly: true, UseStateForUnknown: true,
		Description: "The `pulp_href` (used as the resource identifier).",
	}
}

// labelsField is shared by remotes, repositories and distributions.
func labelsField() field {
	return field{
		Name: "pulp_labels", Kind: fieldLabels,
		Optional: true, Computed: true,
		Description: "Key/value labels.",
	}
}

// variantFields are the two attributes that select the Pulp endpoint. Their
// accepted values come from the featureSet. Both require replacement: a
// change means a different object at a different href.
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

// variantResourceFields prefixes a resource's own attributes with pulp_href
// and the content_type/plugin_name pair.
func variantResourceFields(f featureSet, own ...field) []field {
	return slices.Concat([]field{hrefField()}, variantFields(f), own)
}
