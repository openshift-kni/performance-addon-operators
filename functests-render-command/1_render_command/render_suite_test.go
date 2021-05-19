package __render_command

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	testDir      string
	workspaceDir string
	binPath      string
)

func TestRenderCmd(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Render Command Suite")
}

var _ = BeforeSuite(func() {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		Fail("Cannot retrieve test directory")
	}

	testDir = filepath.Dir(file)
	workspaceDir = filepath.Clean(filepath.Join(testDir, "..", ".."))
	binPath = filepath.Clean(filepath.Join(workspaceDir, "build", "_output", "bin"))
	fmt.Fprintf(GinkgoWriter, "using binary at %q\n", binPath)
})
