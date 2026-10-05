// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"sort"
	"strings"
)

// Pulp serves one endpoint per variant of a resource kind, at
// /pulp/api/v3/<kind>/<content_type>/<plugin_name>/, and each accepts a
// different set of fields.
//
// A featureSet records which optional attributes each variant supports. It
// drives the OneOf validators, the check that a content_type/plugin_name
// combination exists, and whether an attribute reaches a request body.
// Supporting a new plugin is one entry here.
//
// The entries mirror the POST bodies of the Pulp 3 OpenAPI schema;
// TestFeatureSetsMatchAPISchema checks them against a vendored copy.
type featureSet map[string]map[string]bool

// Feature names are the Pulp field names.
const (
	featureAllowUploads  = "allow_uploads"
	featureCaCertificate = "ca_certificate"
	featureContentGuard  = "content_guard"
	featureDistributions = "distributions"
	featureGuards        = "guards"
	featureHeaderName    = "header_name"
	featureHeaderValue   = "header_value"
	featureJqFilter      = "jq_filter"
	featureNamespace     = "namespace"
	featurePolicy        = "policy"
	featurePrivate       = "private"
	featureRemote        = "remote"
)

func variantKey(contentType, pluginName string) string {
	return contentType + "/" + pluginName
}

// supports reports whether a variant accepts an attribute. An unknown variant
// supports nothing.
func (f featureSet) supports(contentType, pluginName, feature string) bool {
	return f[variantKey(contentType, pluginName)][feature]
}

// knows reports whether Pulp serves an endpoint for the variant.
func (f featureSet) knows(contentType, pluginName string) bool {
	_, ok := f[variantKey(contentType, pluginName)]
	return ok
}

func (f featureSet) contentTypes() []string { return f.keyHalves(0) }

func (f featureSet) pluginNames() []string { return f.keyHalves(1) }

func (f featureSet) keyHalves(i int) []string {
	seen := make(map[string]struct{}, len(f))
	for k := range f {
		seen[strings.SplitN(k, "/", 2)[i]] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for s := range seen {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// variants returns every combination, sorted.
func (f featureSet) variants() []string {
	out := make([]string, 0, len(f))
	for k := range f {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// variantsMarkdown renders the combinations for a schema description.
func (f featureSet) variantsMarkdown() string {
	variants := f.variants()
	quoted := make([]string, len(variants))
	for i, v := range variants {
		quoted[i] = fmt.Sprintf("`%s`", v)
	}
	return strings.Join(quoted, ", ")
}

// variantsWith renders the variants that support a feature.
func (f featureSet) variantsWith(feature string) string {
	var out []string
	for _, v := range f.variants() {
		if f[v][feature] {
			out = append(out, fmt.Sprintf("`%s`", v))
		}
	}
	return strings.Join(out, ", ")
}

// distributionFeatures omits content_guard, repository, repository_version
// and pulp_labels, which every variant accepts. namespace is read-only and
// only reported by container distributions.
var distributionFeatures = featureSet{
	"ansible/ansible":           {},
	"container/container":       {featurePrivate: true, featureNamespace: true},
	"container/pull-through":    {featurePrivate: true, featureNamespace: true, featureDistributions: true, featureRemote: true},
	"core/openpgp":              {},
	"deb/apt":                   {},
	"file/file":                 {},
	"gem/gem":                   {featureRemote: true},
	"hugging_face/hugging-face": {featureRemote: true},
	"maven/maven":               {featureRemote: true},
	"npm/npm":                   {featureRemote: true},
	"ostree/ostree":             {},
	"python/pypi":               {featureRemote: true, featureAllowUploads: true},
	"rpm/rpm":                   {},
}

// remoteFeatures: every variant accepts url, tls_validation, username,
// password and pulp_labels; only the git-backed ones lack a policy.
var remoteFeatures = featureSet{
	"ansible/collection":        {featurePolicy: true},
	"ansible/git":               {},
	"ansible/role":              {featurePolicy: true},
	"container/container":       {featurePolicy: true},
	"container/pull-through":    {featurePolicy: true},
	"deb/apt":                   {featurePolicy: true},
	"file/file":                 {featurePolicy: true},
	"file/git":                  {},
	"gem/gem":                   {featurePolicy: true},
	"hugging_face/hugging-face": {featurePolicy: true},
	"maven/maven":               {featurePolicy: true},
	"npm/npm":                   {featurePolicy: true},
	"ostree/ostree":             {featurePolicy: true},
	"python/python":             {featurePolicy: true},
	"rpm/rpm":                   {featurePolicy: true},
	"rpm/uln":                   {featurePolicy: true},
}

// repositoryFeatures: every variant accepts the attributes this provider
// exposes, so the map only drives variant validation.
var repositoryFeatures = featureSet{
	"ansible/ansible":           {},
	"container/container":       {},
	"core/openpgp_keyring":      {},
	"deb/apt":                   {},
	"file/file":                 {},
	"gem/gem":                   {},
	"hugging_face/hugging-face": {},
	"maven/maven":               {},
	"npm/npm":                   {},
	"ostree/ostree":             {},
	"python/python":             {},
	"rpm/rpm":                   {},
}

// contentGuardFeatures: every variant accepts name and description.
var contentGuardFeatures = featureSet{
	"certguard/rhsm":        {featureCaCertificate: true},
	"certguard/x509":        {featureCaCertificate: true},
	"core/composite":        {featureGuards: true},
	"core/content_redirect": {},
	"core/header":           {featureHeaderName: true, featureHeaderValue: true, featureJqFilter: true},
	"core/rbac":             {},
}
