/*
 * Copyright (c) 2023 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/herdstat/herdstat.
 *
 * SPDX-License-Identifier: MIT
 */

package internal

import (
	"github.com/araddon/dateparse"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Computing the previous Sunday", func() {
	When("given a Sunday", func() {
		It("returns that same day", func() {
			sunday := dateparse.MustParse("2023-01-15")
			Expect(previousSunday(sunday)).To(Equal(sunday))
		})
	})
	When("given days that are not Sundays", func() {
		It("returns the last Sunday before that date", func() {
			sunday := dateparse.MustParse("2023-01-08")
			for i := 0; i < 7; i++ {
				day := dateparse.MustParse("2023-01-14").AddDate(0, 0, -i)
				Expect(previousSunday(day)).To(Equal(sunday))
			}
		})
	})
})

var _ = Describe("Computing the number of days between two days", func() {
	When("given exactly the same date", func() {
		It("returns 0", func() {
			day := dateparse.MustParse("2023-01-15")
			Expect(DaysBetween(day, day)).To(Equal(0))
		})
	})
	When("given the same day with a different hour", func() {
		It("returns 0", func() {
			a := dateparse.MustParse("2023-01-15")
			b := a.Add(23 * time.Hour)
			Expect(DaysBetween(a, b)).To(Equal(0))
		})
	})
	When("given two different days", func() {
		It("returns the number of days between these to days", func() {
			a := dateparse.MustParse("2023-01-15")
			b := a.AddDate(0, 1, 2)
			Expect(DaysBetween(a, b)).To(Equal(33))
		})
	})
})
