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

package irqs

import (
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"k8s.io/kubernetes/pkg/kubelet/cm/cpuset"

	"github.com/openshift-kni/debug-tools/pkg/fswrap"
)

const (
	EffectiveAffinity = 1 << iota
)

type Info struct {
	Source string
	IRQ    int
	CPUs   cpuset.CPUSet
}

type Handler struct {
	log        *log.Logger
	procfsRoot string
	fs         fswrap.FSWrapper
}

func New(logger *log.Logger, procfsRoot string) *Handler {
	return &Handler{
		log:        logger,
		procfsRoot: procfsRoot,
		fs:         fswrap.FSWrapper{Log: logger},
	}
}

func (handler *Handler) ReadInfo(flags uint) ([]Info, error) {
	// the best source of information here is man 5 procfs
	// and https://www.kernel.org/doc/Documentation/IRQ-affinity.txt
	irqRoot := filepath.Join(handler.procfsRoot, "irq")

	files, err := handler.fs.ReadDir(irqRoot)
	if err != nil {
		return nil, err
	}

	var irqs []int
	for _, file := range files {
		irq, err := strconv.Atoi(file.Name())
		if err != nil {
			continue // just skip not-irq-looking dirs
		}
		irqs = append(irqs, irq)
	}

	sort.Ints(irqs)

	affinityListFile := "smp_affinity_list"
	if (flags & EffectiveAffinity) == EffectiveAffinity {
		affinityListFile = "effective_affinity_list"
	}

	irqInfos := make([]Info, len(irqs))
	for _, irq := range irqs {
		irqDir := filepath.Join(irqRoot, fmt.Sprintf("%d", irq))

		irqCpuList, err := handler.fs.ReadFile(filepath.Join(irqDir, affinityListFile))
		if err != nil {
			return nil, err
		}

		irqCpus, err := cpuset.Parse(strings.TrimSpace(string(irqCpuList)))
		if err != nil {
			handler.log.Printf("Error parsing cpulist in %q: %v", irqCpuList, err)
			continue // keep running
		}

		irqInfos = append(irqInfos, Info{
			CPUs:   irqCpus,
			IRQ:    irq,
			Source: handler.findSourceForIRQ(irq),
		})
	}
	return irqInfos, nil
}

// TODO: we may want to crosscorrelate with `/proc/interrupts, which always give a valid (!= "") source
func (handler *Handler) findSourceForIRQ(irq int) string {
	irqDir := filepath.Join(handler.procfsRoot, "irq", fmt.Sprintf("%d", irq))
	files, err := handler.fs.ReadDir(irqDir)
	if err != nil {
		handler.log.Printf("Error reading %q: %v", irqDir, err)
		return "MISSING"
	}
	for _, file := range files {
		if file.IsDir() {
			return file.Name()
		}
	}
	handler.log.Printf("Cannot find source for irq %d", irq)
	return ""
}
