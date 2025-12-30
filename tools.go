//go:build tools
// +build tools

// Package tools tracks build-time dependencies for this module.
// This file is used to ensure that tools like terraform-plugin-docs
// are included in go.mod even though they're only used during build/generate.
package tools

import (
	_ "github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs"
)






