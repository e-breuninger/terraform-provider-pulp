// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// distributionResource reaches the hooks on the embedded pulpResource.
func distributionResource(t *testing.T) pulpResource[PulpDistributionModel] {
	t.Helper()
	r, ok := NewPulpDistributionResource().(*pulpDistributionResource)
	if !ok {
		t.Fatal("NewPulpDistributionResource did not return *pulpDistributionResource")
	}
	return r.pulpResource
}

func pullThroughModel(contentType, pluginName string) *PulpDistributionModel {
	return &PulpDistributionModel{
		ContentType: types.StringValue(contentType),
		PluginName:  types.StringValue(pluginName),
		BasePath:    types.StringValue("8f1d7e1e-0d0e-4f1a-9b4c-1f2a3b4c5d6e"),
	}
}

// Pulp returns a generated UUID in base_path and the real registry path in a
// label, so the label is what the configuration asked for.
func TestAfterHydrateReportsPullThroughRegistryPath(t *testing.T) {
	r := distributionResource(t)
	model := pullThroughModel("container", "pull-through")
	data := map[string]any{
		"base_path": model.BasePath.ValueString(),
		"pulp_labels": map[string]any{
			pullThroughDistributionLabel: "docker",
			"team":                       "oct",
		},
	}

	r.afterHydrate(context.Background(), data, model)

	if got := model.BasePath.ValueString(); got != "docker" {
		t.Errorf("base_path = %q, want %q", got, "docker")
	}
	labels := model.PulpLabels.Elements()
	if _, ok := labels[pullThroughDistributionLabel]; ok {
		t.Errorf("%s leaked into pulp_labels: %v", pullThroughDistributionLabel, labels)
	}
	if _, ok := labels["team"]; !ok {
		t.Errorf("unrelated label was dropped: %v", labels)
	}
}

// A distribution of any other variant reports base_path verbatim.
func TestAfterHydrateLeavesOtherVariantsAlone(t *testing.T) {
	r := distributionResource(t)
	model := pullThroughModel("maven", "maven")
	model.BasePath = types.StringValue("maven")
	data := map[string]any{
		"base_path":   "maven",
		"pulp_labels": map[string]any{pullThroughDistributionLabel: "docker"},
	}

	r.afterHydrate(context.Background(), data, model)

	if got := model.BasePath.ValueString(); got != "maven" {
		t.Errorf("base_path = %q, want %q", got, "maven")
	}
}

// A pull-through distribution created before the label existed keeps the
// base_path Pulp reports.
func TestAfterHydrateLeavesUnmarkedPullThroughAlone(t *testing.T) {
	r := distributionResource(t)
	model := pullThroughModel("container", "pull-through")
	model.BasePath = types.StringValue("docker")
	data := map[string]any{"base_path": "docker", "pulp_labels": map[string]any{}}

	r.afterHydrate(context.Background(), data, model)

	if got := model.BasePath.ValueString(); got != "docker" {
		t.Errorf("base_path = %q, want %q", got, "docker")
	}
}

func TestBeforeUpdateOmitsUnchangedPullThroughBasePath(t *testing.T) {
	r := distributionResource(t)

	for _, tc := range []struct {
		name                 string
		contentType, plugin  string
		planPath, statePath  string
		wantBasePathInUpdate bool
	}{
		{
			name:        "unchanged pull-through path is omitted",
			contentType: "container", plugin: "pull-through",
			planPath: "docker", statePath: "docker",
			wantBasePathInUpdate: false,
		},
		{
			// Pulp rejects this itself, with a message naming the repair
			// command, which is more use than silently dropping the change.
			name:        "changed pull-through path is sent",
			contentType: "container", plugin: "pull-through",
			planPath: "docker-hub", statePath: "docker",
			wantBasePathInUpdate: true,
		},
		{
			name:        "other variants always send base_path",
			contentType: "maven", plugin: "maven",
			planPath: "maven", statePath: "maven",
			wantBasePathInUpdate: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan := &PulpDistributionModel{
				ContentType: types.StringValue(tc.contentType),
				PluginName:  types.StringValue(tc.plugin),
				BasePath:    types.StringValue(tc.planPath),
			}
			state := &PulpDistributionModel{
				ContentType: types.StringValue(tc.contentType),
				PluginName:  types.StringValue(tc.plugin),
				BasePath:    types.StringValue(tc.statePath),
			}
			body := map[string]any{"base_path": tc.planPath, "name": "docker"}

			r.beforeUpdate(context.Background(), plan, state, body)

			_, got := body["base_path"]
			if got != tc.wantBasePathInUpdate {
				t.Errorf("base_path present = %v, want %v", got, tc.wantBasePathInUpdate)
			}
			if _, ok := body["name"]; !ok {
				t.Error("beforeUpdate removed an unrelated attribute")
			}
		})
	}
}
