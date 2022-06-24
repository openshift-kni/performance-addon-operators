package hugepages

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	igntypes "github.com/coreos/ignition/v2/config/v3_2/types"
	machineconfigv1 "github.com/openshift/machine-config-operator/pkg/apis/machineconfiguration.openshift.io/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"

	performancev2 "github.com/openshift-kni/performance-addon-operators/api/v2"
	comps "github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components"
	"github.com/openshift-kni/performance-addon-operators/pkg/controller/performanceprofile/components/machineconfig"
)

const (
	defaultIgnitionVersion       = "3.2.0"
	defaultIgnitionContentSource = "data:text/plain;charset=utf-8;base64"
	bashScriptsDir               = "/usr/local/bin"
	hugepagesAllocation          = "hugepages-allocation" //script name
)

//MakeMachineConfig returns machineconfig object based on the hugepages configuration
func MakeMachineConfig(assetsDir string, hugepages *performancev2.HugePages, nodeRole string) (*machineconfigv1.MachineConfig, error) {
	labels := make(map[string]string)
	labels[comps.MachineConfigRoleLabelKey] = nodeRole

	mc := &machineconfigv1.MachineConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: machineconfigv1.GroupVersion.String(),
			Kind:       "MachineConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "hugepages-config",
			Labels: labels,
		},
		Spec: machineconfigv1.MachineConfigSpec{},
	}

	ignitionConfig, err := getIgnitionConfig(assetsDir, hugepages)
	if err != nil {
		return nil, err
	}

	rawIgnition, err := json.Marshal(ignitionConfig)
	if err != nil {
		return nil, err
	}
	mc.Spec.Config = runtime.RawExtension{Raw: rawIgnition}

	return mc, nil
}

func getIgnitionConfig(assetsDir string, hugepages *performancev2.HugePages) (*igntypes.Config, error) {
	ignitionConfig := &igntypes.Config{
		Ignition: igntypes.Ignition{
			Version: defaultIgnitionVersion,
		},
		Storage: igntypes.Storage{
			Files: []igntypes.File{},
		},
	}

	// add hugepages allocation script file under the node /usr/local/bin directory
	mode := 0700
	src := filepath.Join(assetsDir, "scripts", fmt.Sprintf("%s.sh", hugepagesAllocation))
	if err := machineconfig.AddFile(ignitionConfig, src, machineconfig.GetBashScriptPath(hugepagesAllocation), &mode); err != nil {
		return nil, err
	}

	// add hugepages units under systemd
	for _, page := range hugepages.Pages {
		hugepagesSize, err := machineconfig.GetHugepagesSizeKilobytes(page.Size)
		if err != nil {
			return nil, err
		}

		hugepagesService, err := machineconfig.GetSystemdContent(machineconfig.GetHugepagesAllocationUnitOptions(
			hugepagesSize,
			page.Count,
			*page.Node,
		))
		if err != nil {
			return nil, err
		}

		ignitionConfig.Systemd.Units = append(ignitionConfig.Systemd.Units, igntypes.Unit{
			Contents: &hugepagesService,
			Enabled:  pointer.BoolPtr(true),
			Name:     machineconfig.GetSystemdService(fmt.Sprintf("%s-%skB-NUMA%d", hugepagesAllocation, hugepagesSize, *page.Node)),
		})
	}

	return ignitionConfig, nil
}
