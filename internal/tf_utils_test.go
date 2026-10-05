// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package internal

import (
	"math/big"
	"testing"
)

func TestNumberOrNullMatchesConfigPrecision(t *testing.T) {
	for _, s := range []string{"0.1", "2.5", "3", "300"} {
		v, _ := new(big.Float).SetString(s)
		f, _ := v.Float64()
		want, _, _ := big.ParseFloat(s, 10, 512, big.ToNearestEven)

		got := NumberOrNull(map[string]any{"n": f}, "n").ValueBigFloat()
		if got.Cmp(want) != 0 {
			t.Errorf("%s: got %s, want %s", s, got.Text('g', -1), want.Text('g', -1))
		}
	}
}
