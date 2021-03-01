package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jaypipes/ghw/pkg/snapshot"
)

func main() {
	debug := flag.Bool("debug", false, "enable debug output")
	output := flag.String("output", "", "path to clone system information into")
	rootDir := flag.String("root", "", "pseudofs root - use this if running inside a container")
	dumpList := flag.Bool("dump", false, "just dump the glob list of expected content and exit")
	// use this to debug container behaviour
	sleepTime := flag.Int("sleep", 0, "amount of seconds to sleep once done, before exit")
	flag.Parse()

	// ghw can't handle duplicates in CopyFilesInto, the operation will fail.
	// Hence we need to make sure we just don't feed duplicates.
	fileSpecs := dedupExpectedContent(kniExpectedCloneContent(), snapshot.ExpectedCloneContent())
	if *dumpList {
		for _, fileSpec := range fileSpecs {
			fmt.Printf("%s\n", fileSpec)
		}
		os.Exit(0)
	}

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

	if *sleepTime > 0 {
		log.Printf("sleeping for %d seconds before exit", *sleepTime)
		time.Sleep(time.Duration(*sleepTime) * time.Second)
	}
}

func chrootFileSpecs(fileSpecs []string, root string) []string {
	var entries []string
	for _, fileSpec := range fileSpecs {
		entries = append(entries, filepath.Join(root, fileSpec))
	}
	return entries
}

func dedupExpectedContent(fileSpecs, extraFileSpecs []string) []string {
	specSet := make(map[string]int)
	for _, fileSpec := range fileSpecs {
		specSet[fileSpec]++
	}
	for _, extraFileSpec := range extraFileSpecs {
		specSet[extraFileSpec]++
	}

	var retSpecs []string
	for retSpec := range specSet {
		retSpecs = append(retSpecs, retSpec)
	}
	return retSpecs
}

func kniExpectedCloneContent() []string {
	return []string{
		// generic information
		"/proc/cmdline",
		// IRQ affinities
		"/proc/interrupts",
		"/proc/irq/default_smp_affinity",
		"/proc/irq/*/*affinity_list",
		"/proc/irq/*/node",
		// KNI-specific CPU infos:
		"/sys/devices/system/cpu/smt/active",
		// BIOS/firmware versions
		"/sys/class/dmi/id/bios*",
		"/sys/class/dmi/id/product_family",
		"/sys/class/dmi/id/product_name",
		"/sys/class/dmi/id/product_sku",
		"/sys/class/dmi/id/product_version",
	}
}
