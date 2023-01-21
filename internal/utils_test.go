/*
 * Copyright (c) 2023 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/herdstat/herdstat.
 *
 * SPDX-License-Identifier: MIT
 */

package internal

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Taking elements", func() {
	Context("from a non-empty slice", func() {
		s := make([]int, 2)
		s[0] = 1
		s[1] = 2

		When("taking no element", func() {
			It("returns an empty slice and the original slice", func() {
				first, rest := take(s, 0)
				Expect(first).To(HaveLen(0))
				Expect(rest).To(HaveLen(2))
				Expect(rest[0]).To(Equal(1))
				Expect(rest[1]).To(Equal(2))
			})
		})
		When("taking 1 element", func() {
			It("returns the first element and the rest of the slice", func() {
				first, rest := take(s, 1)
				Expect(first).To(HaveLen(1))
				Expect(first[0]).To(Equal(1))
				Expect(rest).To(HaveLen(1))
				Expect(rest[0]).To(Equal(2))
			})
		})
		When("taking all elements", func() {
			It("returns all elements and an empty slice", func() {
				first, rest := take(s, len(s))
				Expect(first).To(HaveLen(len(s)))
				Expect(first[0]).To(Equal(1))
				Expect(first[1]).To(Equal(2))
				Expect(rest).To(HaveLen(0))
			})
		})
	})
})

var _ = Describe("Finding the maximum element", func() {
	Context("from a non-empty array", func() {
		comp := func(a, b int) int {
			return a - b
		}
		When("there is a single maximum element", func() {
			arr := []int{1, 2, 4, 2}
			It("returns an this element", func() {
				Expect(max(arr, comp)).To(Equal(4))
			})
		})
		When("there are multiple maximum elements", func() {
			arr := []int{1, 3, 3, 2}
			It("returns on of these element", func() {
				Expect(max(arr, comp)).To(Equal(3))
			})
		})
	})
})

var _ = Describe("Getting the keys of a map", func() {
	When("the map is not empty", func() {
		m := map[int]int{
			1: 101,
			2: 102,
		}
		It("returns the keys", func() {
			Expect(Keys(m)).To(Equal([]int{1, 2}))
		})
	})
	When("the map is empty", func() {
		m := map[int]int{}
		It("returns an empty array", func() {
			Expect(Keys(m)).To(Equal([]int{}))
		})
	})
})
