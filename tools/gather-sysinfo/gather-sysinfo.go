package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"

	"github.com/jaypipes/ghw/pkg/snapshot"
)

func main() {
	debug := flag.Bool("debug", false, "enable debug output")
	output := flag.String("output", "", "path to clone system information into")
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

	if err := snapshot.CloneTreeInto(scratchDir); err != nil {
		log.Fatalf("error cloning into %q: %v", scratchDir, err)
	}

	// extra, KNI-specific entries
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
	}
	if err := snapshot.CopyFilesInto(fileSpecs, scratchDir, nil); err != nil {
		log.Fatalf("error cloning extra files into %q: %v", scratchDir, err)
	}

	if err := snapshot.PackFrom(*output, scratchDir); err != nil {
		log.Fatalf("error packing %q into %q: %v", scratchDir, *output, err)
	}
}
