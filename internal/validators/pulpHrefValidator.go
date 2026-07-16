// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package validator

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// PulpHrefRegex matches "/pulp/api/<version>/", e.g. "/pulp/api/v3/" or
// "/pulp/api/v4/". Pulp's API surface is expected to stay the same across
// versions - only the version segment of the path changes.
var PulpHrefRegex = regexp.MustCompile(`^\/pulp\/api\/v\d+\/`)

type PulpHrefValidatorType struct{}

func PulpHrefValidator() validator.String {
	return PulpHrefValidatorType{}
}

func (v PulpHrefValidatorType) Description(ctx context.Context) string {
	return "must start with /pulp/api/v<version>/, e.g. /pulp/api/v3/"
}

func (v PulpHrefValidatorType) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v PulpHrefValidatorType) ValidateString(
	ctx context.Context,
	req validator.StringRequest,
	resp *validator.StringResponse,
) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	value := req.ConfigValue.ValueString()

	if !PulpHrefRegex.MatchString(value) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid pulp_href value",
			fmt.Sprintf(
				"Value %q must start with %q.",
				value, PulpHrefRegex.String(),
			),
		)
	}
}
