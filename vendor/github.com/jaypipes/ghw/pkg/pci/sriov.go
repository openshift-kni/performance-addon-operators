//
// Use and distribution licensed under the Apache license version 2.
//
// See the COPYING file in the root project directory for full text.
//

package pci

type VFInfo struct {
	ID         int    `json:"id"`
	PCIAddress string `json:"pci_address"`
}

type SRIOVPhysFn struct {
	MaxVFNum int      `json:"max_vf_num"`
	VFs      []VFInfo `json:"vfs"`
}

type SRIOVVirtFn struct {
	ParentPCIAddress string `json:"parent_pci_address"`
}

type SRIOVInfo struct {
	Driver     string       `json:"driver"`
	Interfaces []string     `json:"interfaces"`
	PhysFn     *SRIOVPhysFn `json:"physfn,omitempty"`
	VirtFn     *SRIOVVirtFn `json:"virtfn,omitempty"`
}
