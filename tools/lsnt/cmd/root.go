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

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

type cmdOpts struct {
	sysFSRoot string
	verbose   int
}

var opts cmdOpts

// NewRootCommand returns entrypoint command to interact with all other commands
func NewRootCommand() *cobra.Command {

	root := &cobra.Command{
		Use:   "lsnt",
		Short: "lsnt displays *N*UMA *T*opology informations about the system",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprint(cmd.OutOrStderr(), cmd.UsageString())
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.Flags().StringVarP(&opts.sysFSRoot, "sysfs", "S", "/sys", "sysfs root")
	root.Flags().IntVar(&opts.verbose, "verbose", 1, "verbosiness level")

	root.AddCommand(
		newCPUCommand(),
		newNUMACommand(),
		newPCIDevsCommand(),
		newDaemonWaitCommand(),
	)

	return root

}
