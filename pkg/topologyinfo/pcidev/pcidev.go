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

package pcidev

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	// PathBusPCIDevices is the subpath which holds informations about the PCI(-express) devices
	PathBusPCIDevices = "bus/pci/devices/"
)

// PCIDeviceInfo represents the information about a single PCI(-express) device
type PCIDeviceInfo interface {
	// Address is the FULL PCI address (bus_id:device_id) of the device, as string
	Address() string
	// String returns human-friendly representation of the device info
	String() string
	// DevClass is the PCI device class, as integer
	DevClass() int64
	// Vendor is the PCI vendor identifier, as integer
	Vendor() int64
	// Device is the PCI device identifier, as integer
	Device() int64
	// TODO: driver
}

// PCIDeviceInfoList is a list of PCIDeviceInfo
type PCIDeviceInfoList []PCIDeviceInfo

// PCIDevices reports the information about all the PCI(-express) devices found in the system
type PCIDevices struct {
	NUMAPCIDevices map[int]PCIDeviceInfoList
}

// NewPCIDevices extracts the information about the PCI(-express) devices from a given sysfs-like path
func NewPCIDevices(sysfs string) (*PCIDevices, error) {
	sysfsPath := filepath.Join(sysfs, PathBusPCIDevices)

	numaNodePCIDevs := make(map[int]PCIDeviceInfoList)
	entries, err := ioutil.ReadDir(sysfsPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		isPhysFn := false
		isVFn := false
		numVfs := 0
		parentFn := ""
		numvfsPath := filepath.Join(sysfsPath, entry.Name(), "sriov_numvfs")
		if _, err := os.Stat(numvfsPath); err == nil {
			isPhysFn = true
			numVfs, _ = readInt(numvfsPath)
		} else if !os.IsNotExist(err) {
			// unexpected error. Bail out
			return nil, err
		}
		physFnPath := filepath.Join(sysfsPath, entry.Name(), "physfn")
		if _, err := os.Stat(physFnPath); err == nil {
			isVFn = true
			if dest, err := os.Readlink(physFnPath); err == nil {
				parentFn = filepath.Base(dest)
			}
		} else if !os.IsNotExist(err) {
			// unexpected error. Bail out
			return nil, err
		}

		devPath := filepath.Join(sysfsPath, entry.Name())
		nodeNum, err := readInt(filepath.Join(devPath, "numa_node"))
		// FIX for single-numa node (e.g. dev laptop)
		if nodeNum < 0 {
			nodeNum = 0
		}

		devClass, err := readHexInt64(filepath.Join(devPath, "class"))
		if err != nil {
			return nil, err
		}
		vendor, err := readHexInt64(filepath.Join(devPath, "vendor"))
		if err != nil {
			return nil, err
		}
		device, err := readHexInt64(filepath.Join(devPath, "device"))
		if err != nil {
			return nil, err
		}

		pciDevs := numaNodePCIDevs[nodeNum]
		pciDevs = append(pciDevs, SRIOVDeviceInfo{
			IsPhysFn: isPhysFn,
			NumVFS:   numVfs,
			IsVFn:    isVFn,
			ParentFn: parentFn,
			address:  entry.Name(),
			numaNode: nodeNum,
			devClass: devClass,
			vendor:   vendor,
			device:   device,
		})
		numaNodePCIDevs[nodeNum] = pciDevs
	}

	return &PCIDevices{
		NUMAPCIDevices: numaNodePCIDevs,
	}, nil

}

// SRIOVDeviceInfo extends PCIDeviceInfo with SRIOV-specific data
type SRIOVDeviceInfo struct {
	// ISPhysFn is true if this device is a SRIOV PHYSical FunctioN
	IsPhysFn bool
	// NumVFS is the NUMber of Virtual Functions this device have configured, if IsPhysFn=true. Meaningless otherwise
	NumVFS int // only PFs
	// IsVFn is true if this device is a Virtual FunctioN
	IsVFn bool
	// ParentFn is the bus_id:device_id PCI(-express) address of the parent Physical Function, if IsVFn=true. Meaningless otherwise.
	ParentFn string // only VFs
	address  string
	numaNode int
	devClass int64
	vendor   int64
	device   int64
}

// Address is the FULL PCI address (bus_id:device_id) of the device, as string
func (sdi SRIOVDeviceInfo) Address() string {
	return sdi.address
}

// DevClass is the PCI device class, as integer
func (sdi SRIOVDeviceInfo) DevClass() int64 {
	return sdi.devClass
}

// Vendor is the PCI vendor identifier, as integer
func (sdi SRIOVDeviceInfo) Vendor() int64 {
	return sdi.vendor
}

// Device is the PCI device identifier, as integer
func (sdi SRIOVDeviceInfo) Device() int64 {
	return sdi.device
}

func (sdi SRIOVDeviceInfo) String() string {
	return fmt.Sprintf("pci@%s %x:%x numa_node=%d physfn=%v vfn=%v", sdi.address, sdi.vendor, sdi.device, sdi.numaNode, sdi.IsPhysFn, sdi.IsVFn)
}

func readHexInt64(path string) (int64, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(strings.TrimSpace(string(content)), 0, 64)
}

func readInt(path string) (int, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(content)))
}
