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

package runtime

import (
	"fmt"

	"multi-cri/pkg/cri/adapters"
	"multi-cri/pkg/cri/store"
	"net"

	"golang.org/x/net/context"
	k8snet "k8s.io/apimachinery/pkg/util/net"
	runtimeApi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
	"k8s.io/kubernetes/pkg/kubelet/server/streaming"
)

// instrumentedService wraps service and logs each operation.
type MulticriRuntime struct {
	*multicriService
}

func NewMulticriRuntime(c *multicriService) CRIMulticriService {
	return &MulticriRuntime{multicriService: c}
}

func (r *MulticriRuntime) Version(_ context.Context, req *runtimeApi.VersionRequest) (*runtimeApi.VersionResponse, error) {
	return r.adapter.Version()
}

func (r *MulticriRuntime) Status(_ context.Context, req *runtimeApi.StatusRequest) (*runtimeApi.StatusResponse, error) {
	runtimeCondition := &runtimeApi.RuntimeCondition{
		Type:   runtimeApi.RuntimeReady,
		Status: true,
	}
	networkCondition := &runtimeApi.RuntimeCondition{
		Type:   runtimeApi.NetworkReady,
		Status: true,
	}
	if r.netPlugin != nil {
		if err := r.netPlugin.Status(); err != nil {
			networkCondition.Status = false
			networkCondition.Reason = "NetworkPluginNotReady"
			networkCondition.Message = fmt.Sprintf("Network plugin returns error: %v", err)
		}
	}
	return &runtimeApi.StatusResponse{
		Status: &runtimeApi.RuntimeStatus{Conditions: []*runtimeApi.RuntimeCondition{
			runtimeCondition,
			networkCondition,
		}},
	}, nil
}

func (r *MulticriRuntime) UpdateRuntimeConfig(_ context.Context, req *runtimeApi.UpdateRuntimeConfigRequest) (*runtimeApi.UpdateRuntimeConfigResponse, error) {
	return &runtimeApi.UpdateRuntimeConfigResponse{}, nil
}

func NewStreamServer(a adapters.AdapterInterface, c store.ContainerStoreInterface, addr, port string) (streaming.Server, error) {
	if addr == "" {
		a, err := k8snet.ChooseBindAddress(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get stream server address: %v", err)
		}
		addr = a.String()
	}
	config := streaming.DefaultConfig
	config.Addr = net.JoinHostPort(addr, port)
	runtime := a.NewStreamRuntime(c)
	return streaming.NewServer(config, runtime)
}
