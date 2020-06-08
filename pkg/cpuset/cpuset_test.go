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
	"reflect"
	"testing"
)

func TestEmpty(t *testing.T) {
	checkEmpty(t, Empty())
}

func TestParseEmpty(t *testing.T) {
	cpus, err := Parse("")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	checkEmpty(t, cpus)
}

func TestParseRange(t *testing.T) {
	testCases := []tcase{
		{"0,1,2,3", []int{0, 1, 2, 3}},
		// we don't elide dupes
		{"0,1,1,3", []int{0, 1, 1, 3}},
	}
	for _, testCase := range testCases {
		testCase.CheckParseSlices(t)
	}
}

var cpuSetTestCases []tcase = []tcase{
	{"", []int{}},
	{"1", []int{1}},
	{"0-7", []int{0, 1, 2, 3, 4, 5, 6, 7}},
	{"0-3,5-7", []int{0, 1, 2, 3, 5, 6, 7}},
	{"0,2-7", []int{0, 2, 3, 4, 5, 6, 7}},
	{"0,4,7-15", []int{0, 4, 7, 8, 9, 10, 11, 12, 13, 14, 15}},
	{"0-5,7", []int{0, 1, 2, 3, 4, 5, 7}},
}

func TestParseRangeInterval(t *testing.T) {
	for _, testCase := range cpuSetTestCases {
		testCase.CheckParseSlices(t)
	}
}

func TestUnparseRangeInterval(t *testing.T) {
	for _, testCase := range cpuSetTestCases {
		testCase.CheckUnparseSlices(t)
	}
}

func TestParseUnparseRangeInterval(t *testing.T) {
	for _, testCase := range cpuSetTestCases {
		testCase.CheckParseUnparseSlices(t)
	}
}

func TestParseRangeIntervalMalformed(t *testing.T) {
	testCases := []tcase{
		{",", nil},
		{"-", nil},
		{"-,", nil},
		{",-", nil},
		{",-,", nil},
		{",,", nil},
		{"1,2-,6", nil},
		{"1,-3,6", nil},
		{"1,2-4-6,8", nil},
		{"1,-,8", nil},
	}
	for _, testCase := range testCases {
		testCase.ExpectError(t)
	}
}

func mustParse(t *testing.T, s string) []int {
	cpus, err := Parse(s)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	return cpus
}

func checkEmpty(t *testing.T, cpus []int) {
	if cpus == nil {
		t.Errorf("empty must not be nil")
	}
	if len(cpus) != 0 {
		t.Errorf("empty must have zero length")
	}
}

type tcase struct {
	s string
	v []int
}

func (tc tcase) CheckParseSlices(t *testing.T) {
	cpus := mustParse(t, tc.s)
	if !reflect.DeepEqual(cpus, tc.v) {
		t.Errorf("slices differ: expected=%v cpus=%v", tc.v, cpus)
	}
}

func (tc tcase) CheckUnparseSlices(t *testing.T) {
	cpus := Unparse(tc.v)
	if tc.s != cpus {
		t.Errorf("strings differ: expected=%q cpus=%q", tc.s, cpus)
	}
}

func (tc tcase) CheckParseUnparseSlices(t *testing.T) {
	cpus := mustParse(t, tc.s)
	cpuString := Unparse(cpus)
	if cpuString != tc.s {
		t.Errorf("parse/unparse mismatch: expected=%s cpuString=%s cpus=%q", tc.s, cpuString, cpus)
	}
}

func (tc tcase) ExpectError(t *testing.T) {
	_, err := Parse(tc.s)
	if err == nil {
		t.Errorf("unexpectedly ok: %q", tc.s)
	}
}
