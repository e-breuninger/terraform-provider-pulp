// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// apiSchemaPath is where Pulp serves its own OpenAPI schema. The conformance
// tests read it from the Pulp the acceptance tests run against, so the feature
// maps are always checked against the version actually in use rather than a
// copy that goes stale in the repository. They skip when no Pulp is reachable.
const apiSchemaPath = "/pulp/api/v3/docs/api.json"

// openAPI is the slice of the schema these tests need.
type openAPI struct {
	Paths map[string]map[string]struct {
		RequestBody struct {
			Content map[string]struct {
				Schema jsonSchema `json:"schema"`
			} `json:"content"`
		} `json:"requestBody"`
		Responses map[string]struct {
			Content map[string]struct {
				Schema jsonSchema `json:"schema"`
			} `json:"content"`
		} `json:"responses"`
	} `json:"paths"`
	Components struct {
		Schemas map[string]jsonSchema `json:"schemas"`
	} `json:"components"`
}

type jsonSchema struct {
	Ref        string                `json:"$ref"`
	ReadOnly   bool                  `json:"readOnly"`
	Properties map[string]jsonSchema `json:"properties"`
	Items      *jsonSchema           `json:"items"`
}

func (a *openAPI) deref(s jsonSchema) jsonSchema {
	for range 10 {
		if s.Ref == "" {
			return s
		}
		name := s.Ref[strings.LastIndex(s.Ref, "/")+1:]
		next, ok := a.Components.Schemas[name]
		if !ok {
			return jsonSchema{}
		}
		s = next
	}
	return s
}

// The schema is a few megabytes, so it is fetched once per test binary.
var (
	apiSchemaOnce sync.Once
	apiSchema     *openAPI
	apiSchemaErr  error
)

func loadAPISchema(t *testing.T) *openAPI {
	t.Helper()

	apiSchemaOnce.Do(func() { apiSchema, apiSchemaErr = fetchAPISchema() })
	if apiSchemaErr != nil {
		t.Skipf("no Pulp to read the API schema from: %v", apiSchemaErr)
	}
	return apiSchema
}

func fetchAPISchema() (*openAPI, error) {
	serverURL := os.Getenv("PULP_SERVER_URL")
	if serverURL == "" {
		serverURL = defaultTestServerURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL+apiSchemaPath, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: status %d", apiSchemaPath, resp.StatusCode)
	}

	var api openAPI
	if err := json.NewDecoder(resp.Body).Decode(&api); err != nil {
		return nil, fmt.Errorf("decoding the API schema: %w", err)
	}
	return &api, nil
}

// variantFields reports, per variant, the fields its POST body accepts and
// the fields its listing returns.
func (a *openAPI) variantFields(collection string) (writable, readable map[string]map[string]bool) {
	pattern := regexp.MustCompile(`^/pulp/api/v3/` + collection + `/([^/{]+)/([^/{]+)/$`)
	writable = map[string]map[string]bool{}
	readable = map[string]map[string]bool{}

	for path, operations := range a.Paths {
		match := pattern.FindStringSubmatch(path)
		if match == nil {
			continue
		}
		variant := variantKey(match[1], match[2])

		if post, ok := operations["post"]; ok {
			body := a.deref(post.RequestBody.Content["application/json"].Schema)
			fields := map[string]bool{}
			for name, prop := range body.Properties {
				if !prop.ReadOnly && !a.deref(prop).ReadOnly {
					fields[name] = true
				}
			}
			writable[variant] = fields
		}

		if get, ok := operations["get"]; ok {
			// A listing wraps the objects in
			// {count, next, previous, results: [ <object> ]}.
			page := a.deref(get.Responses["200"].Content["application/json"].Schema)
			results := a.deref(page.Properties["results"])
			fields := map[string]bool{}
			if results.Items != nil {
				for name := range a.deref(*results.Items).Properties {
					fields[name] = true
				}
			}
			readable[variant] = fields
		}
	}
	return writable, readable
}

// TestFeatureSetsMatchAPISchema checks every declared variant and gated
// attribute against Pulp's own schema, so a Pulp upgrade that adds or drops a
// plugin fails here.
func TestFeatureSetsMatchAPISchema(t *testing.T) {
	api := loadAPISchema(t)

	for _, tc := range []struct {
		collection string
		features   featureSet
		// tracked features are compared against the schema.
		tracked []string
		// readOnly features live in the response, not the POST body.
		readOnly []string
	}{
		{
			collection: "distributions",
			features:   distributionFeatures,
			tracked:    []string{featureRemote, featurePrivate, featureDistributions, featureAllowUploads, featureNamespace},
			readOnly:   []string{featureNamespace},
		},
		{
			collection: "remotes",
			features:   remoteFeatures,
			tracked:    []string{featurePolicy},
		},
		{
			collection: "repositories",
			features:   repositoryFeatures,
		},
		{
			collection: "contentguards",
			features:   contentGuardFeatures,
			tracked:    []string{featureCaCertificate, featureGuards, featureHeaderName, featureHeaderValue, featureJqFilter},
		},
	} {
		t.Run(tc.collection, func(t *testing.T) {
			writable, readable := api.variantFields(tc.collection)
			if len(writable) == 0 {
				t.Fatalf("no %s endpoints found in the API schema", tc.collection)
			}

			t.Run("variants", func(t *testing.T) {
				want := sortedKeys(writable)
				got := tc.features.variants()
				if !slices.Equal(got, want) {
					t.Errorf("declared variants do not match Pulp:\n  declared: %v\n  Pulp:     %v\n  missing:  %v\n  extra:    %v",
						got, want, missing(want, got), missing(got, want))
				}
			})

			for _, feature := range tc.tracked {
				t.Run(feature, func(t *testing.T) {
					source := writable
					if slices.Contains(tc.readOnly, feature) {
						source = readable
					}

					var want []string
					for variant, fields := range source {
						if fields[feature] {
							want = append(want, variant)
						}
					}
					slices.Sort(want)

					var got []string
					for _, variant := range tc.features.variants() {
						if tc.features[variant][feature] {
							got = append(got, variant)
						}
					}

					// The variants subtest above covers undeclared ones.
					want = slices.DeleteFunc(want, func(v string) bool {
						return !slices.Contains(tc.features.variants(), v)
					})

					if !slices.Equal(got, want) {
						t.Errorf("variants supporting %q do not match Pulp:\n  declared: %v\n  Pulp:     %v",
							feature, got, want)
					}
				})
			}
		})
	}
}

// TestGatedAttributesAreRealPulpFields checks every attribute is spelled the
// way Pulp spells it; the field table relies on that.
func TestGatedAttributesAreRealPulpFields(t *testing.T) {
	api := loadAPISchema(t)

	for name, r := range declaredResources(t) {
		table, ok := r.(fieldTable)
		if !ok {
			continue
		}
		ft, ok := r.(featureTable)
		if !ok || ft.featureTable() == nil {
			continue
		}

		t.Run(name, func(t *testing.T) {
			writable, readable := api.variantFields(ft.collectionName())

			for _, f := range table.fieldTable() {
				if f.Local {
					continue
				}
				known := false
				for _, fields := range []map[string]map[string]bool{writable, readable} {
					for _, names := range fields {
						if names[f.Name] {
							known = true
						}
					}
				}
				if !known {
					t.Errorf("attribute %q is not a field of any %s variant in the Pulp API",
						f.Name, ft.collectionName())
				}
			}
		})
	}
}

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

// missing returns the elements of want absent from got.
func missing(want, got []string) []string {
	out := []string{}
	for _, w := range want {
		if !slices.Contains(got, w) {
			out = append(out, w)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
