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

	"github.com/openshift-kni/performance-addon-operators/pkg/cpuset"
	"github.com/openshift-kni/performance-addon-operators/pkg/topologyinfo/cpus"
)

func showNUMA(cmd *cobra.Command, args []string) error {
	cpuRes, err := cpus.NewCPUs(opts.sysFSRoot)
	if err != nil {
		return err
	}

	sys := gotree.New(".")
	for nodeID, cpuIDList := range cpuRes.NUMANodeCPUs {
		numaNode := sys.Add(fmt.Sprintf("numa%02d", nodeID))
		numaNode.Add(cpuset.Unparse(cpuIDList))
	}
	fmt.Println(sys.Print())
	return nil
}

func newNUMACommand() *cobra.Command {
	show := &cobra.Command{
		Use:   "numa",
		Short: "show NUMA device tree",
		RunE:  showNUMA,
		Args:  cobra.NoArgs,
	}
	return show
}
