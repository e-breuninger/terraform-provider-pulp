// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package validators

import (
	"context"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// NumberAtLeast rejects numbers below min. terraform-plugin-framework-validators
// has no numbervalidator equivalent.
func NumberAtLeast(min int64) validator.Number {
	return numberAtLeastValidator{min: big.NewFloat(float64(min))}
}

type numberAtLeastValidator struct{ min *big.Float }

func (v numberAtLeastValidator) Description(context.Context) string {
	return fmt.Sprintf("must be at least %s", v.min.Text('f', -1))
}

func (v numberAtLeastValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v numberAtLeastValidator) ValidateNumber(
	ctx context.Context,
	req validator.NumberRequest,
	resp *validator.NumberResponse,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if value := req.ConfigValue.ValueBigFloat(); value.Cmp(v.min) < 0 {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid number value",
			fmt.Sprintf("Value %s %s.", value.Text('f', -1), v.Description(ctx)))
	}
}
