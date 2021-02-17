package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/jaypipes/ghw/pkg/snapshot"
)

func main() {
	debug := flag.Bool("debug", false, "enable debug output")
	output := flag.String("output", "", "path to clone system information into")
	rootDir := flag.String("root", "", "pseudofs root - use this if running inside a container")
	flag.Parse()

	if *output == "" {
		log.Fatal("--output is required")
	}

	if *debug {
		snapshot.SetTraceFunction(func(msg string, args ...interface{}) {
			log.Printf(msg, args...)
		})
	}

	scratchDir, err := ioutil.TempDir("", "perf-must-gather-*")
	if err != nil {
		log.Fatalf("error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(scratchDir)

	// collect only KNI-specific entries
	fileSpecs := []string{
		// generic information
		"/proc/cmdline",
		// IRQ affinities
		"/proc/interrupts",
		"/proc/irq/default_smp_affinity",
		"/proc/irq/*/*affinity_list",
		"/proc/irq/*/node",
		// BIOS/firmware versions
		"/sys/class/dmi/id/bios*",
		"/sys/class/dmi/id/product_family",
		"/sys/class/dmi/id/product_name",
		"/sys/class/dmi/id/product_sku",
		"/sys/class/dmi/id/product_version",
		// basic memory infos
		"/proc/meminfo",
		// PCI device data
		"/sys/bus/pci/devices/*",
		"/sys/devices/pci*/*/irq",
		"/sys/devices/pci*/*/local_cpulist",
		"/sys/devices/pci*/*/modalias",
		"/sys/devices/pci*/*/numa_node",
		"/sys/devices/pci*/pci_bus/*/cpulistaffinity",
		// CPU topology
		"/sys/devices/system/cpu/cpu*/cache/index*/*",
		"/sys/devices/system/cpu/cpu*/topology/*",
		"/sys/devices/system/memory/block_size_bytes",
		"/sys/devices/system/memory/memory*/online",
		"/sys/devices/system/memory/memory*/state",
		// NUMA topology
		"/sys/devices/system/node/has_*",
		"/sys/devices/system/node/online",
		"/sys/devices/system/node/possible",
		"/sys/devices/system/node/node*/cpu*",
		"/sys/devices/system/node/node*/distance",
	}

	if *rootDir != "" {
		fileSpecs = chrootFileSpecs(fileSpecs, *rootDir)
	}

	if err := snapshot.CopyFilesInto(fileSpecs, scratchDir, nil); err != nil {
		log.Fatalf("error cloning extra files into %q: %v", scratchDir, err)
	}

	if *rootDir != "" {
		scratchDir = filepath.Join(scratchDir, *rootDir)
	}

	dest := *output
	if dest == "-" {
		err = snapshot.PackWithWriter(os.Stdout, scratchDir)
		dest = "stdout"
	} else {
		err = snapshot.PackFrom(dest, scratchDir)
	}
	if err != nil {
		log.Fatalf("error packing %q to %q: %v", scratchDir, dest, err)
	}
}

func chrootFileSpecs(fileSpecs []string, root string) []string {
	var entries []string
	for _, fileSpec := range fileSpecs {
		entries = append(entries, filepath.Join(root, fileSpec))
	}
	return entries
}
