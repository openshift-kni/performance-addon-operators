//go:build main
// +build main

package main

// A dummy go file that will be ignored for builds, but included for dependencies.
import (
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
