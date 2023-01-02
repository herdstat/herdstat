/*
 * Copyright (c) 2023 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/herdstat/herdstat.
 *
 * SPDX-License-Identifier: MIT
 */

package internal

import "time"

// previousSunday returns the last Sunday before the given date. If the given
// date is a Sunday, the date is returned unaltered.
func previousSunday(date time.Time) time.Time {
	return date.AddDate(0, 0, -int(date.Weekday()))
}

// DaysBetween computes the number of days between two days.
func DaysBetween(a, b time.Time) int {
	return int(b.Sub(a).Hours() / 24)
}
