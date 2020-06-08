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
	"strings"
)

// Unparse takes a cpuset as (unsorted) slice of ints and returns a representing cpuset definition
func Unparse(v []int) string {
	cpus := sorted(v)
	num := len(cpus)

	if num == 0 {
		return ""
	}
	if num == 1 {
		return fmt.Sprintf("%d", cpus[0])
	}

	makeAtom := func(cpus []int, begin, end int) string {
		if begin < (end - 1) { // range
			return fmt.Sprintf("%d-%d", cpus[begin], cpus[end-1])
		}
		return fmt.Sprintf("%d", cpus[begin])
	}

	var atoms []string
	begin := 0 // of the potential range
	end := 1   // of the potential range
	for end < num {
		if (cpus[end] - cpus[end-1]) > 1 { // seam
			atoms = append(atoms, makeAtom(cpus, begin, end))
			begin = end
		}
		end++
	}
	// collect reminder
	if begin < end {
		atoms = append(atoms, makeAtom(cpus, begin, end))

	}
	return strings.Join(atoms, ",")
}

func sorted(v []int) []int {
	r := make([]int, len(v))
	copy(r, v)
	sort.Ints(r)
	return r
}
