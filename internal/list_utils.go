// Copyright (c) E. Breuninger GmbH & Co
// SPDX-License-Identifier: MPL-2.0

package internal

import "sort"

// Diff returns (toAdd, toRemove) between desired and current slices.
func Diff(desired, current []string) (toAdd, toRemove []string) {
	d := map[string]struct{}{}
	for _, s := range desired {
		d[s] = struct{}{}
	}
	c := map[string]struct{}{}
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
