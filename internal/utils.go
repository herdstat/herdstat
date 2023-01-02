/*
 * Copyright (c) 2023 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/herdstat/herdstat.
 *
 * SPDX-License-Identifier: MIT
 */

package internal

// take first n elements from the given slice and return slices for the taken
// elements and the remaining elements.
func take[T any](records []T, n int) ([]T, []T) {
	return records[:n], records[n:]
}

// max returns the maximum of the elements according to the given comparison
// function.
func max[T any](elements []T, comp func(a, b T) int) T {
	var max T
	for i, e := range elements {
		if i == 0 || comp(e, max) > 0 {
			max = e
		}
	}
	return max
}

// Keys returns the array of keys of the given map.
func Keys[K comparable, V any](m map[K]V) []K {
	keys := make([]K, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	return keys
}
