// +build tools

// Official workaround to track tool dependencies with go modules:
// https://github.com/golang/go/wiki/Modules#how-can-i-track-tool-dependencies-for-a-module

package tools

import (
	// code needed by client generator
	_ "k8s.io/code-generator/cmd/client-gen"
)
