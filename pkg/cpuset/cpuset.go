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

package cpuset

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Empty returns a empty cpuset, a slice of ints
func Empty() []int {
	return []int{}
}

// Parse takess a string representing a cpuset definition, and returns as sorted slice of ints
func Parse(s string) ([]int, error) {
	cpus := Empty()
	if s == "" {
		return cpus, nil
	}

	// the cpuset is a comma-separated list of items
	for _, item := range strings.Split(s, ",") {
		// each item could be either a constant ('1') or a interval ('0-2')
		interval := strings.Split(item, "-")
		if len(interval) == 1 {
			// it's a constant
			cpu, err := strconv.Atoi(interval[0])
			if err != nil {
				return cpus, err
			}
			cpus = append(cpus, cpu)
		} else if len(interval) == 2 {
			// it's a real interval: a range
			start, err := strconv.Atoi(interval[0])
			if err != nil {
				return cpus, err
			}
			stop, err := strconv.Atoi(interval[1])
			if err != nil {
				return cpus, err
			}
			for cpu := start; cpu <= stop; cpu++ {
				cpus = append(cpus, cpu)
			}
		} else {
			return cpus, fmt.Errorf("malformed interval: %q", interval)
		}
	}

	sort.Ints(cpus)
	return cpus, nil
}
