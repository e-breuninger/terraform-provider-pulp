// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package internal

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/helper/acctest"
)

// hrefVariantParts is the segment count of a plugin-typed pulp_href:
// /pulp/api/v3/repositories/npm/npm/<uuid>/.
const hrefVariantParts = 7

// ParseHrefVariant extracts the content_type and plugin_name from a
// plugin-typed pulp_href.
func ParseHrefVariant(pulpHref string) (contentType, pluginName string, err error) {
	parts := strings.Split(strings.Trim(pulpHref, "/"), "/")
	if len(parts) < hrefVariantParts {
		return "", "", fmt.Errorf(
			"could not parse content_type and plugin_name from pulp_href %q: expected at least %d path segments, got %d (%v)",
			pulpHref, hrefVariantParts, len(parts), parts)
	}
	return parts[4], parts[5], nil
}

// CompositeID joins a pulp_href and a role into an import ID, for resources
// Pulp gives no href of their own.
func CompositeID(cgHref, role string) string {
	return fmt.Sprintf("%s|%s", cgHref, role)
}

// SplitCompositeID is the inverse of CompositeID.
func SplitCompositeID(id string) (cgHref, role string, err error) {
	parts := strings.SplitN(id, "|", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid composite ID %q, expected `<contentguard_href>|<role>`", id)
	}
	return parts[0], parts[1], nil
}

// RandomSuffix keeps acceptance test fixtures from colliding.
func RandomSuffix() string {
	return acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum)
}
