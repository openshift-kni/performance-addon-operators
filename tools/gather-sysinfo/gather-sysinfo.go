package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/jaypipes/ghw/pkg/snapshot"
	"github.com/openshift-kni/debug-tools/pkg/knit/cmd"
	"github.com/spf13/cobra"
)

type snapshotOptions struct {
	dumpList  bool
	output    string
	rootDir   string
	sleepTime int
}

func main() {
	root := cmd.NewRootCommand(newSnapshotCommand)

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func newSnapshotCommand(knitOpts *cmd.KnitOptions) *cobra.Command {
	opts := &snapshotOptions{}
	snap := &cobra.Command{
		Use:   "snapshot",
		Short: "snapshot pseudofilesystems for offline analysis",
		RunE: func(cmd *cobra.Command, args []string) error {
			return makeSnapshot(cmd, knitOpts, opts, args)
		},
		Args: cobra.NoArgs,
	}
	snap.Flags().StringVar(&opts.rootDir, "root", "", "pseudofs root - use this if running inside a container")
	snap.Flags().StringVar(&opts.output, "output", "", "path to clone system information into")
	snap.Flags().BoolVar(&opts.dumpList, "dump", false, "just dump the glob list of expected content and exit")
	// use this to debug container behaviour
	snap.Flags().IntVar(&opts.sleepTime, "sleep", 0, "amount of seconds to sleep once done, before exit")

	return snap
}

func makeSnapshot(cmd *cobra.Command, knitOpts *cmd.KnitOptions, opts *snapshotOptions, args []string) error {
	// ghw can't handle duplicates in CopyFilesInto, the operation will fail.
	// Hence we need to make sure we just don't feed duplicates.
	fileSpecs := dedupExpectedContent(kniExpectedCloneContent(), snapshot.ExpectedCloneContent())
	if opts.dumpList {
		for _, fileSpec := range fileSpecs {
			fmt.Printf("%s\n", fileSpec)
		}
		return nil
	}

	if opts.output == "" {
		return fmt.Errorf("--output is required")
	}

	if knitOpts.Debug {
		snapshot.SetTraceFunction(func(msg string, args ...interface{}) {
			knitOpts.Log.Printf(msg, args...)
		})
	}

	scratchDir, err := ioutil.TempDir("", "perf-must-gather-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(scratchDir)

	if opts.rootDir != "" {
		fileSpecs = chrootFileSpecs(fileSpecs, opts.rootDir)
	}

	if err := snapshot.CopyFilesInto(fileSpecs, scratchDir, nil); err != nil {
		return fmt.Errorf("error cloning extra files into %q: %v", scratchDir, err)
	}

	if opts.rootDir != "" {
		scratchDir = filepath.Join(scratchDir, opts.rootDir)
	}

	dest := opts.output
	if dest == "-" {
		err = snapshot.PackWithWriter(os.Stdout, scratchDir)
		dest = "stdout"
	} else {
		err = snapshot.PackFrom(dest, scratchDir)
	}
	if err != nil {
		return fmt.Errorf("error packing %q to %q: %v", scratchDir, dest, err)
	}

	if opts.sleepTime > 0 {
		knitOpts.Log.Printf("sleeping for %d seconds before exit", opts.sleepTime)
		time.Sleep(time.Duration(opts.sleepTime) * time.Second)
	}

	return nil
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
