// Copyright (c) 2019 Atrio, Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package remote

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"k8s.io/klog"
	runtimeApi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
	"k8s.io/kubernetes/pkg/kubelet/util"
)

const (
	ConnectionTimeoutSeconds = 20

	maxMsgSize              = 1024 * 1024 * 16
	MulticriRuntimeHandler = "multicri"
)

type RemoteCRIObject struct {
	runtimeApi.RuntimeServiceClient
	runtimeApi.ImageServiceClient
}

type RemoteCRIConfiguration struct {
	remoteCRIList map[string]*RemoteCRIObject
}

// NewRemoteRuntimeService creates a new runtimeApi.RuntimeServiceClient.
func newRemoteRuntimeClient(endpoint string, connectionTimeout time.Duration) (runtimeApi.RuntimeServiceClient, error) {
	klog.V(3).Infof("Connecting to runtime service %s", endpoint)
	addr, dailer, err := util.GetAddressAndDialer(endpoint)
	if err != nil {
		return nil, err
	}
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithTimeout(connectionTimeout), grpc.WithDialer(dailer), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMsgSize)))
	if err != nil {
		klog.Errorf("Connect remote runtime %s failed: %v", addr, err)
		return nil, err
	}

	return runtimeApi.NewRuntimeServiceClient(conn), nil
}

func newRemoteImageClient(endpoint string, connectionTimeout time.Duration) (runtimeApi.ImageServiceClient, error) {
	klog.V(3).Infof("Connecting to image service %s", endpoint)
	addr, dailer, err := util.GetAddressAndDialer(endpoint)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure(), grpc.WithDialer(dailer), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMsgSize)))
	if err != nil {
		klog.Errorf("Connect remote image service %s failed: %v", addr, err)
		return nil, err
	}

	return runtimeApi.NewImageServiceClient(conn), nil
}

func getRemoteCRI(remoteRuntimeEndpoint, remoteImageEndpoint string, durantion time.Duration) (*RemoteCRIObject, error) {
	rs, err := newRemoteRuntimeClient(remoteRuntimeEndpoint, durantion)
	if err != nil {
		return nil, err
	}
	is, err := newRemoteImageClient(remoteImageEndpoint, durantion)
	if err != nil {
		return nil, err
	}

	return &RemoteCRIObject{rs, is}, err
}

func LoadRemoteRuntimeConfiguration(config string) (*RemoteCRIConfiguration, error) {
	list := make(map[string]*RemoteCRIObject)
	if config != "" {
		runtimePlugins := strings.Split(config, ",")

		for _, plugin := range runtimePlugins {
			pluginSplit := strings.Split(plugin, ":")
			if len(pluginSplit) != 2 {
				return nil, fmt.Errorf("Bad format for remote runtime. %s ", plugin)
			}
			cri, err := getRemoteCRI(pluginSplit[1], pluginSplit[1], time.Duration(time.Second*ConnectionTimeoutSeconds))
			if err != nil {
				return nil, err
			}
			list[pluginSplit[0]] = cri
		}
	}
	return &RemoteCRIConfiguration{list}, nil
}

func (r RemoteCRIConfiguration) GetRemoteRuntime(runtimeHandlerName string) (*RemoteCRIObject, error) {
	if runtimeHandlerName == MulticriRuntimeHandler || len(r.remoteCRIList) == 0 {
		return nil, nil
	}
	if runtimeHandlerName == "" {
		runtimeHandlerName = "default"
	}
	if remoteRuntime, ok := r.remoteCRIList[runtimeHandlerName]; ok {
		return remoteRuntime, nil
	}
	return nil, fmt.Errorf("RemoteRuntime %s not found", runtimeHandlerName)
}

func (r RemoteCRIConfiguration) MulticriRuntimeName() string {
	if _, ok := r.remoteCRIList["default"]; ok {
		return MulticriRuntimeHandler
	}
	return ""
}
