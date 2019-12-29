package machineconfigpool

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFeatureGate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Machine Config Pool Suite")
}
