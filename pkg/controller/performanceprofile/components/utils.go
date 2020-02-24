package components

import (
	"bytes"
	"fmt"
	"math/big"
	"strconv"
	"strings"
)

const maxSystemCpus = 64

// GetComponentName returns the component name for the specific performance profile
func GetComponentName(profileName string, prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, profileName)
}

// GetFirstKeyAndValue return the first key / value pair of a map
func GetFirstKeyAndValue(m map[string]string) (string, string) {
	for k, v := range m {
		return k, v
	}
	return "", ""
}

// SplitLabelKey returns the given label key splitted up in domain and role
func SplitLabelKey(s string) (domain, role string, err error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("Can't split %s", s)
	}
	return parts[0], parts[1], nil
}

func cpusListToArray(cpusListStr string) ([]int, error) {
	var cpusList []int
	elements := strings.Split(cpusListStr, ",")
	for _, item := range elements {
		cpuRange := strings.Split(item, "-")
		// provided a range: 1-3
		if len(cpuRange) > 1 {
			start, err := strconv.Atoi(cpuRange[0])
			if err != nil {
				return nil, err
			}
			end, err := strconv.Atoi(cpuRange[1])
			if err != nil {
				return nil, err
			}
			// Add cpus to the list. Assuming it's a valid range.
			for cpuNum := start; cpuNum <= end; cpuNum++ {
				cpusList = append(cpusList, cpuNum)
			}
		} else {
			cpuNum, err := strconv.Atoi(cpuRange[0])
			if err != nil {
				return nil, err
			}
			cpusList = append(cpusList, cpuNum)
		}
	}

	return cpusList, nil
}

// CPUListToHexMask converts a list of cpus into a cpu mask represented in hexdecimal
func CPUListToHexMask(cpulist string) (hexMask string, err error) {
	reservedCpus, err := cpusListToArray(cpulist)
	if err != nil {
		return "", err
	}
	currMask := big.NewInt(0)
	for _, cpu := range reservedCpus {
		x := new(big.Int).Lsh(big.NewInt(1), uint(cpu))
		currMask.Or(currMask, x)
	}
	return fmt.Sprintf("%064x", currMask), nil
}

// CPUListToInvertedMask converts a list of cpus into an inverted cpu mask represented in hexdecimal
func CPUListToInvertedMask(cpulist string) (hexMask string, err error) {
	reservedCpus, err := cpusListToArray(cpulist)
	if err != nil {
		return "", err
	}
	reservedCpusLookup := make(map[int]bool)
	for _, cpu := range reservedCpus {
		reservedCpusLookup[cpu] = true
	}

	currMask := big.NewInt(0)
	for cpu := 0; cpu < maxSystemCpus; cpu++ {
		if _, reserved := reservedCpusLookup[cpu]; reserved {
			continue
		}
		x := new(big.Int).Lsh(big.NewInt(1), uint(cpu))
		currMask.Or(currMask, x)
	}
	return fmt.Sprintf("%016x", currMask), nil
}

// CPUListTo64BitsMaskList converts a list of cpus into an inverted cpu mask represented
// in a list of 64bit hexadecimal mask devided by a delimiter ","
func CPUListTo64BitsMaskList(cpulist string) (hexMask string, err error) {
	maskStr, err := CPUListToInvertedMask(cpulist)
	if err != nil {
		return "", nil
	}
	return fmt.Sprintf("%s,%s", maskStr[:8], maskStr[8:]), nil
}

// CPUListToMaskList converts a list of cpus into a cpu mask represented
// in a list of hexadecimal mask devided by a delimiter ","
func CPUListToMaskList(cpulist string) (hexMask string, err error) {
	maskStr, err := CPUListToHexMask(cpulist)
	if err != nil {
		return "", nil
	}
	index := 0
	for index < (len(maskStr) - 8) {
		if maskStr[index:index+8] != "00000000" {
			break
		}
		index = index + 8
	}
	var b bytes.Buffer
	for index <= (len(maskStr) - 16) {
		b.WriteString(maskStr[index : index+8])
		b.WriteString(",")
		index = index + 8
	}
	b.WriteString(maskStr[index : index+8])
	trimmedCPUMaskList := b.String()
	return trimmedCPUMaskList, nil
}
