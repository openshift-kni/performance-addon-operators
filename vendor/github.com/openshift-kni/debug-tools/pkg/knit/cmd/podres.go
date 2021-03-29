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
	"context"
	"encoding/json"
	"os"
	"time"

	"github.com/spf13/cobra"

	kubeletpodresourcesv1 "k8s.io/kubelet/pkg/apis/podresources/v1"
	"k8s.io/kubernetes/pkg/kubelet/apis/podresources"
)

// see k/k/test/e2e_node/util.go
// TODO: make these options
const (
	defaultSocketPath = "unix:///var/lib/kubelet/pod-resources/kubelet.sock"

	defaultPodResourcesTimeout = 10 * time.Second
	defaultPodResourcesMaxSize = 1024 * 1024 * 16 // 16 Mb
)

type podResOptions struct {
	socketPath string
}

func NewPodResourcesCommand(knitOpts *KnitOptions) *cobra.Command {
	opts := &podResOptions{}
	podRes := &cobra.Command{
		Use:   "podres",
		Short: "show currently allocated pod resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showPodResources(cmd, opts, args)
		},
		Args: cobra.NoArgs,
	}
	podRes.Flags().StringVarP(&opts.socketPath, "socket-path", "R", defaultSocketPath, "podresources API socket path.")
	return podRes
}

func showPodResources(cmd *cobra.Command, opts *podResOptions, args []string) error {
	cli, conn, err := podresources.GetV1Client(opts.socketPath, defaultPodResourcesTimeout, defaultPodResourcesMaxSize)
	if err != nil {
		return err
	}
	defer conn.Close()

	resp, err := cli.List(context.TODO(), &kubeletpodresourcesv1.ListPodResourcesRequest{})
	if err != nil {
		return err
	}

	json.NewEncoder(os.Stdout).Encode(resp)

	return nil
}
