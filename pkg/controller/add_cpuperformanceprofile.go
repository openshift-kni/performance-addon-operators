package controller

import (
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/cpuperformanceprofile"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, cpuperformanceprofile.Add)
}
