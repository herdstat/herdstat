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
