package pao_mustgather

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/nodes"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/profiles"
	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"
	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
)

const destDir = "must-gather"

var _ = Describe("[rfe_id: 50649] Performance Addon Operator Must Gather", func() {
	mgContentFolder := ""

	testutils.BeforeAll(func() {
		destDirContent, err := ioutil.ReadDir(destDir)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "unable to read contents from destDir:%s. error: %w", destDir, err)

		for _, content := range destDirContent {
			if !content.IsDir() {
				continue
			}
			mgContentFolder = filepath.Join(destDir, content.Name())
		}
	})

	Context("with a freshly executed must-gather command", func() {
		It("Verify Generic cluster resource definitions are captured", func() {

			var genericFiles = []string{
				"version",
				"cluster-scoped-resources/config.openshift.io/featuregates/cluster.yaml",
				"namespaces/openshift-cluster-node-tuning-operator/tuned.openshift.io/tuneds/default.yaml",
				"namespaces/openshift-cluster-node-tuning-operator/tuned.openshift.io/tuneds/rendered.yaml",
			}

			By(fmt.Sprintf("Checking Folder: %q\n", mgContentFolder))
			By("\tLooking for generic files")
			err := checkfilesExist(genericFiles, mgContentFolder)
			Expect(err).ToNot(gomega.HaveOccurred())
		})

		It("Verify PAO cluster resources are captured", func() {
			profile, _ := profiles.GetByNodeLabels(testutils.NodeSelectorLabels)
			if profile == nil {
				Skip("No Performance Profile found")
			}
			//replace peformance.yaml for profile.Name when data is generated in the node
			ClusterSpecificFiles := []string{
				"cluster-scoped-resources/performance.openshift.io/performanceprofiles/performance.yaml",
				"cluster-scoped-resources/machineconfiguration.openshift.io/kubeletconfigs/performance-performance.yaml",
				"namespaces/openshift-cluster-node-tuning-operator/tuned.openshift.io/tuneds/openshift-node-performance-performance.yaml",
			}

			By(fmt.Sprintf("Checking Folder: %q\n", mgContentFolder))
			By("\tLooking for generic files")
			err := checkfilesExist(ClusterSpecificFiles, mgContentFolder)
			Expect(err).ToNot(gomega.HaveOccurred())
		})

		It("Verify hardware related information are captured", func() {
			nodeSpecificFiles := []string{
				"proc/cpuinfo",
				"cpu_affinities.json",
				"dmesg",
				"irq_affinities.json",
				"lscpu",
				"machineinfo.json",
				"podresources.json",
				"proc_cmdline",
				"sysinfo.log",
			}

			var workerRTNodes []corev1.Node

			workerRTNodes, err := nodes.GetByLabels(testutils.NodeSelectorLabels)
			Expect(err).ToNot(HaveOccurred())

			workerRTNodes, err = nodes.MatchingOptionalSelector(workerRTNodes)
			Expect(err).ToNot(HaveOccurred())
			cnfWorkerNode := workerRTNodes[0].ObjectMeta.Name

			// find the path of sysinfo.tgz of the correct node
			snapShotName := ""
			err = filepath.Walk(mgContentFolder,
				func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if !info.IsDir() && info.Name() == "sysinfo.tgz" {
						if strings.Contains(path, cnfWorkerNode) {
							snapShotName = path
						}
					}
					return nil
				})
			if err != nil {
				log.Println(err)
			}

			snapShotPath := filepath.Dir(snapShotName)

			err = Untar(snapShotPath, snapShotName)
			Expect(err).ToNot(HaveOccurred(), "failed to read the %s: %v", snapShotName, err)

			err = checkfilesExist(nodeSpecificFiles, snapShotPath)
			Expect(err).ToNot(gomega.HaveOccurred())
		})

		It("Verify machineconfig resources are captured", func() {
			mcps := &machineconfigv1.MachineConfigPoolList{}
			err := testclient.Client.List(context.TODO(), mcps)
			Expect(err).ToNot(HaveOccurred())
			mcpFiles := make([]string, len(mcps.Items))
			for _, item := range mcps.Items {
				mcpFiles = append(mcpFiles, fmt.Sprintf("cluster-scoped-resources/machineconfiguration.openshift.io/machineconfigpools/%s.yaml", item.Name))
			}
			err = checkfilesExist(mcpFiles, mgContentFolder)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

func checkfilesExist(listOfFiles []string, path string) error {
	for _, f := range listOfFiles {
		file := filepath.Join(path, f)
		if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func Untar(root string, snapshotName string) error {
	var err error
	r, err := os.Open(snapshotName)
	if err != nil {
		return err
	}
	defer r.Close()

	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		target := filepath.Join(root, header.Name)
		mode := os.FileMode(header.Mode)

		switch header.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(target, mode)
			if err != nil {
				return err
			}

		case tar.TypeReg:
			dst, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, mode)
			if err != nil {
				return err
			}

			_, err = io.Copy(dst, tr)
			if err != nil {
				return err
			}

			dst.Close()
		}
	}
}
