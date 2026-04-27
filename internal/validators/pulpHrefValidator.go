// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package validator

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var PulpHrefRegex = regexp.MustCompile(`^\/pulp\/api\/v3\/`)

type PulpHrefValidatorType struct{}

func PulpHrefValidator() validator.String {
	return PulpHrefValidatorType{}
}

func (v PulpHrefValidatorType) Description(ctx context.Context) string {
	return "must start with /pulp/api/v3/"
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
