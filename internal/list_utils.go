// Copyright E. Breuninger GmbH & Co 2026
// SPDX-License-Identifier: MPL-2.0

package internal

import "sort"

// Diff returns (toAdd, toRemove) between desired and current slices.
func Diff(desired, current []string) (toAdd, toRemove []string) {
	d := make(map[string]struct{}, len(desired))
	for _, s := range desired {
		d[s] = struct{}{}
	}
	c := make(map[string]struct{}, len(current))
	for _, s := range current {
		c[s] = struct{}{}
	}
	for s := range d {
		if _, ok := c[s]; !ok {
			toAdd = append(toAdd, s)
		}
	}
	for s := range c {
		if _, ok := d[s]; !ok {
			toRemove = append(toRemove, s)
		}
	}
	sort.Strings(toAdd)
	sort.Strings(toRemove)
	return
}

// Intersect returns the elements of want that also appear in have, keeping
// the order of want.
func Intersect(want, have []string) []string {
	haveSet := make(map[string]struct{}, len(have))
	for _, s := range have {
		haveSet[s] = struct{}{}
	}
	out := make([]string, 0, len(want))
	for _, s := range want {
		if _, ok := haveSet[s]; ok {
			out = append(out, s)
		}
	}
	return out
}
