// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.
//

package pci

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jaypipes/ghw/pkg/context"
	"github.com/jaypipes/ghw/pkg/linuxpath"
	"github.com/jaypipes/ghw/pkg/util"
)

func getDeviceSriovInfo(ctx *context.Context, address string) *SRIOVInfo {
	paths := linuxpath.New(ctx)
	pciAddr := AddressFromString(address)
	if pciAddr == nil {
		return nil
	}
	devPath := filepath.Join(paths.SysBusPciDevices, pciAddr.String())

	// see: https://doc.dpdk.org/guides/linux_gsg/linux_drivers.html
	driver := ""
	if dest, err := os.Readlink(filepath.Join(devPath, "driver")); err == nil {
		driver = filepath.Base(dest)
	}
	networkNames := findNetworks(ctx, devPath)

	if dest, err := os.Readlink(filepath.Join(devPath, "physfn")); err == nil {
		// it's a virtual function!
		return &SRIOVInfo{
			Driver:     driver,
			Interfaces: networkNames,
			VirtFn: &SRIOVVirtFn{
				ParentPCIAddress: filepath.Base(dest),
			},
		}
	}
	// it's either a physical function or a non-SRIOV device
	if buf, err := ioutil.ReadFile(filepath.Join(devPath, "sriov_totalvfs")); err == nil {
		// it seems a physical function
		maxVFs, err := strconv.Atoi(strings.TrimSpace(string(buf)))
		if err != nil {
			ctx.Warn("error reading sriov_totalvfn for %q: %v", err)
			return nil
		}

		return &SRIOVInfo{
			Driver:     driver,
			Interfaces: networkNames,
			PhysFn: &SRIOVPhysFn{
				MaxVFNum: maxVFs,
				VFs:      findVFsFromPF(ctx, devPath),
			},
		}
	}
	// not a SRIOV device
	return nil
}

func findNetworks(ctx *context.Context, devPath string) []string {
	netPath := filepath.Join(devPath, "net")

	netEntries, err := ioutil.ReadDir(netPath)
	if err != nil {
		ctx.Warn("cannot enumerate network names for %q: %v", devPath, err)
		return nil
	}

	var networks []string
	for _, netEntry := range netEntries {
		networks = append(networks, netEntry.Name())
	}

	return networks
}

func findVFsFromPF(ctx *context.Context, devPath string) []VFInfo {
	numVfs := util.SafeIntFromFile(ctx, filepath.Join(devPath, "sriov_numvfs"))
	if numVfs == -1 {
		return nil
	}

	var vfs []VFInfo
	for vfnIdx := 0; vfnIdx < numVfs; vfnIdx++ {
		virtFn := fmt.Sprintf("virtfn%d", vfnIdx)
		vfnDest, err := os.Readlink(filepath.Join(devPath, virtFn))
		if err != nil {
			ctx.Warn("error reading backing device for virtfn %q physfn %q: %v", virtFn, devPath, err)
			return nil
		}
		vfs = append(vfs, VFInfo{
			ID:         vfnIdx,
			PCIAddress: filepath.Base(vfnDest),
		})
	}
	return vfs
}
