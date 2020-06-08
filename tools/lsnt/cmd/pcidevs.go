/*
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2020 Red Hat, Inc.
 */

package cmd

import (
	"fmt"

	"github.com/disiqueira/gotree"
	"github.com/spf13/cobra"

	"github.com/openshift-kni/performance-addon-operators/pkg/topologyinfo/pcidev"
)

func showPCIDevs(cmd *cobra.Command, args []string) error {
	pciDevs, err := pcidev.NewPCIDevices(opts.sysFSRoot)
	if err != nil {
		return err
	}

	sys := gotree.New(".")
	for nodeID, devInfos := range pciDevs.NUMAPCIDevices {
		numaNode := sys.Add(fmt.Sprintf("numa%02d", nodeID))
		for _, devInfo := range devInfos {
			extra := fmt.Sprintf(" (%x)", devInfo.DevClass())
			if sriovInfo, ok := devInfo.(pcidev.SRIOVDeviceInfo); ok && (sriovInfo.IsPhysFn || sriovInfo.IsVFn) {
				if sriovInfo.IsPhysFn {
					extra = fmt.Sprintf(" physfn numvfs=%v", sriovInfo.NumVFS)
				} else if sriovInfo.IsVFn {
					extra = fmt.Sprintf(" vfn parent=%s", sriovInfo.ParentFn)
				} else {
					extra = " ???"
				}
			}
			numaNode.Add(fmt.Sprintf("%s %04x:%04x%s", devInfo.Address(), devInfo.Vendor(), devInfo.Device(), extra))
		}
	}
	fmt.Println(sys.Print())
	return nil
}

func newPCIDevsCommand() *cobra.Command {
	show := &cobra.Command{
		Use:   "pcidevs",
		Short: "show PCI devices in the system",
		RunE:  showPCIDevs,
		Args:  cobra.NoArgs,
	}
	return show
}
