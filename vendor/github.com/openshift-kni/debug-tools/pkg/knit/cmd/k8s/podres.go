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

package k8s

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/spf13/cobra"

	kubeletpodresourcesv1 "k8s.io/kubelet/pkg/apis/podresources/v1"
	"k8s.io/kubernetes/pkg/kubelet/apis/podresources"

	"github.com/openshift-kni/debug-tools/pkg/knit/cmd"
)

// see k/k/test/e2e_node/util.go
// TODO: make these options
const (
	defaultSocketPath = "unix:///var/lib/kubelet/pod-resources/kubelet.sock"

	defaultPodResourcesTimeout = 10 * time.Second
	defaultPodResourcesMaxSize = 1024 * 1024 * 16 // 16 Mb
)

const (
	apiCallList           = "list"
	apiCallGetAllocatable = "get-allocatable"
)

type podResOptions struct {
	socketPath string
}

func NewPodResourcesCommand(knitOpts *cmd.KnitOptions) *cobra.Command {
	opts := &podResOptions{}
	podRes := &cobra.Command{
		Use:   "podres",
		Short: "show currently allocated pod resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showPodResources(cmd, opts, args)
		},
		Args: cobra.MaximumNArgs(1),
	}
	podRes.Flags().StringVarP(&opts.socketPath, "socket-path", "R", defaultSocketPath, "podresources API socket path.")
	return podRes
}

// we use spew to avoid artifacts due to JSON mashalling. We want to see all the fields.
func selectAction(apiName string) (func(cli kubeletpodresourcesv1.PodResourcesListerClient) error, error) {
	if apiName == apiCallList {
		return func(cli kubeletpodresourcesv1.PodResourcesListerClient) error {
			resp, err := cli.List(context.TODO(), &kubeletpodresourcesv1.ListPodResourcesRequest{})
			if err != nil {
				return err
			}
			spew.Fdump(os.Stdout, resp)
			return nil
		}, nil
	}
	if apiName == apiCallGetAllocatable {
		return func(cli kubeletpodresourcesv1.PodResourcesListerClient) error {
			resp, err := cli.GetAllocatableResources(context.TODO(), &kubeletpodresourcesv1.AllocatableResourcesRequest{})
			if err != nil {
				return err
			}
			spew.Fdump(os.Stdout, resp)
			return nil
		}, nil
	}
	return func(cli kubeletpodresourcesv1.PodResourcesListerClient) error {
		return nil
	}, fmt.Errorf("unknown API %q", apiName)
}

func showPodResources(cmd *cobra.Command, opts *podResOptions, args []string) error {
	apiName := "list"
	if len(args) == 1 {
		apiName = args[0]
	}

	action, err := selectAction(apiName)
	if err != nil {
		return err
	}

	cli, conn, err := podresources.GetV1Client(opts.socketPath, defaultPodResourcesTimeout, defaultPodResourcesMaxSize)
	if err != nil {
		return err
	}
	defer conn.Close()

	return action(cli)
}
