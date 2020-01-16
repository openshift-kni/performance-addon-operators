package profile

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFeatureGate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kubelet Config Suite")
}
