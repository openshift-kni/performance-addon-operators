package memory_manager

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMemoryManager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MemoryManager e2e tests")
}
