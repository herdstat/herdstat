/*
 * Copyright (c) 2023 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/herdstat/herdstat.
 *
 * SPDX-License-Identifier: MIT
 */

// Package main contains the entrypoint into the herdstat CLI implementation.
package main

import (
	"herdstat/cmd"
)

func main() {
	cmd.Execute()
}
