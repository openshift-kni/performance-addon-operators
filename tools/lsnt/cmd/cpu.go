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
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/openshift-kni/performance-addon-operators/pkg/cpuset"
	"github.com/openshift-kni/performance-addon-operators/pkg/topologyinfo/cpus"
)

func summarizeCPUIdList(data map[int]cpus.CPUIdList) string {
	ref := 0
	var items []string
	for cpuID, cpuList := range data {
		cur := len(cpuList)
		if ref == 0 {
			ref = cur
		} else if ref != cur {
			items = append(items, fmt.Sprintf("core%d=%d", cpuID, cur))
		}
	}
	if len(items) > 0 {
		return strings.Join(items, ",")
	}
	return fmt.Sprintf("%d", ref)
}

func summary(cpuRes *cpus.CPUs) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintf(w, "CPU(s):\t%d\n", len(cpuRes.Present))
	fmt.Fprintf(w, "Present CPU(s) list:\t%s\n", cpuset.Unparse(cpuRes.Present))
	fmt.Fprintf(w, "On-line CPU(s) list:\t%s\n", cpuset.Unparse(cpuRes.Online))
	fmt.Fprintf(w, "Thread(s) per core:\t%s\n", summarizeCPUIdList(cpuRes.CoreCPUs))
	fmt.Fprintf(w, "Core(s) per socket:\t%s\n", summarizeCPUIdList(cpuRes.PackageCPUs))
	fmt.Fprintf(w, "Socket(s):\t%d\n", cpuRes.Packages)
	fmt.Fprintf(w, "NUMA node(s):\t%d\n", cpuRes.NUMANodes)
	for i := 0; i < cpuRes.NUMANodes; i++ {
		fmt.Fprintf(w, "NUMA node%d CPU(s):\t%s\n", i, cpuset.Unparse(cpuRes.NUMANodeCPUs[i]))
	}
	w.Flush()

}

func showCPU(cmd *cobra.Command, args []string) error {
	cpuRes, err := cpus.NewCPUs(opts.sysFSRoot)
	if err != nil {
		return err
	}
	summary(cpuRes)
	return nil
}

func newCPUCommand() *cobra.Command {
	show := &cobra.Command{
		Use:   "cpu",
		Short: "show cpu details like lscpu(1)",
		RunE:  showCPU,
		Args:  cobra.NoArgs,
	}
	return show
}
