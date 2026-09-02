// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"sort"
	"strings"
)

// Pulp does not serve one endpoint per resource kind: it serves one endpoint
// per *variant* of a kind, addressed as
// /pulp/api/v3/<kind>/<content_type>/<plugin_name>/. Each variant accepts a
// different set of fields — an npm distribution takes a `remote`, an rpm one
// does not; only python distributions take `allow_uploads`; only
// container ones have `private`. Posting a field the endpoint does not know
// is rejected outright.
//
// A featureSet is the provider's model of that: for one resource kind, which
// optional attributes each variant supports. It is the single source of truth
// behind three things that used to be maintained by hand and had drifted
// apart:
//
//   - the `content_type` and `plugin_name` OneOf validators,
//   - the check that the *combination* of the two is one Pulp actually serves,
//   - the decision to include an attribute in a request body.
//
// Supporting a new Pulp plugin is therefore a single entry here; nothing else
// in the resource needs to change.
//
// The entries mirror the POST request bodies of the Pulp 3 OpenAPI schema.
// TestFeatureSetsMatchAPISchema checks them against a vendored copy of it.
type featureSet map[string]map[string]bool

// Feature names. These are the Pulp field names, and are shared by the schema
// attribute, the request body key and the response key.
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

// supports reports whether the given variant accepts the named attribute.
// An unknown variant supports nothing.
func (f featureSet) supports(contentType, pluginName, feature string) bool {
	return f[variantKey(contentType, pluginName)][feature]
}

// knows reports whether Pulp serves an endpoint for the given variant.
func (f featureSet) knows(contentType, pluginName string) bool {
	_, ok := f[variantKey(contentType, pluginName)]
	return ok
}

// contentTypes returns every distinct content_type, sorted.
func (f featureSet) contentTypes() []string { return f.keyHalves(0) }

// pluginNames returns every distinct plugin_name, sorted.
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

// variants returns every "<content_type>/<plugin_name>" combination, sorted.
func (f featureSet) variants() []string {
	out := make([]string, 0, len(f))
	for k := range f {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// variantsMarkdown renders the supported combinations as a markdown list, for
// use in the schema description so the docs cannot drift from the code.
func (f featureSet) variantsMarkdown() string {
	variants := f.variants()
	quoted := make([]string, len(variants))
	for i, v := range variants {
		quoted[i] = fmt.Sprintf("`%s`", v)
	}
	return strings.Join(quoted, ", ")
}

// variantsWith renders, as a markdown list, the variants that support the
// given feature. Used to tell the practitioner where an attribute they set
// would actually be accepted.
func (f featureSet) variantsWith(feature string) string {
	var out []string
	for _, v := range f.variants() {
		if f[v][feature] {
			out = append(out, fmt.Sprintf("`%s`", v))
		}
	}
	return strings.Join(out, ", ")
}

// distributionFeatures lists the distribution variants Pulp serves.
//
// `content_guard`, `repository`, `repository_version` and `pulp_labels` are
// accepted by every variant and so are not tracked here. `namespace` is
// read-only — it is listed because it is only *reported* by container
// distributions, and is null everywhere else.
var distributionFeatures = featureSet{
	"ansible/ansible":           {},
	"container/container":       {featurePrivate: true, featureNamespace: true},
	"container/pull-through":    {featurePrivate: true, featureNamespace: true, featureDistributions: true, featureRemote: true},
	"core/openpgp":              {},
	"deb/apt":                   {},
	"file/file":                 {},
	"hugging_face/hugging-face": {featureRemote: true},
	"maven/maven":               {featureRemote: true},
	"npm/npm":                   {featureRemote: true},
	"ostree/ostree":             {},
	"python/pypi":               {featureRemote: true, featureAllowUploads: true},
	"rpm/rpm":                   {},
}

// remoteFeatures lists the remote variants Pulp serves. Every variant accepts
// `url`, `tls_validation`, `username`, `password` and `pulp_labels`; only the
// git-backed ones lack a download `policy`.
var remoteFeatures = featureSet{
	"ansible/collection":        {featurePolicy: true},
	"ansible/git":               {},
	"ansible/role":              {featurePolicy: true},
	"container/container":       {featurePolicy: true},
	"container/pull-through":    {featurePolicy: true},
	"deb/apt":                   {featurePolicy: true},
	"file/file":                 {featurePolicy: true},
	"file/git":                  {},
	"hugging_face/hugging-face": {featurePolicy: true},
	"maven/maven":               {featurePolicy: true},
	"npm/npm":                   {featurePolicy: true},
	"ostree/ostree":             {featurePolicy: true},
	"python/python":             {featurePolicy: true},
	"rpm/rpm":                   {featurePolicy: true},
	"rpm/uln":                   {featurePolicy: true},
}

// repositoryFeatures lists the repository variants Pulp serves. The
// attributes this provider exposes (`description`, `remote`, `pulp_labels`)
// are accepted by all of them, so no variant has extra features — the map
// exists for the content_type/plugin_name validation.
var repositoryFeatures = featureSet{
	"ansible/ansible":           {},
	"container/container":       {},
	"core/openpgp_keyring":      {},
	"deb/apt":                   {},
	"file/file":                 {},
	"hugging_face/hugging-face": {},
	"maven/maven":               {},
	"npm/npm":                   {},
	"ostree/ostree":             {},
	"python/python":             {},
	"rpm/rpm":                   {},
}

// contentGuardFeatures lists the content guard variants Pulp serves. Every
// variant accepts `name` and `description`; the rest are variant-specific.
var contentGuardFeatures = featureSet{
	"certguard/rhsm":        {featureCaCertificate: true},
	"certguard/x509":        {featureCaCertificate: true},
	"core/composite":        {featureGuards: true},
	"core/content_redirect": {},
	"core/header":           {featureHeaderName: true, featureHeaderValue: true, featureJqFilter: true},
	"core/rbac":             {},
}
