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

	"golang.org/x/net/context"
	"k8s.io/klog"
	runtimeApi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

func (r *BabelFishRuntime) Attach(ctx context.Context, req *runtimeApi.AttachRequest) (*runtimeApi.AttachResponse, error) {
	containerId := req.ContainerId
	klog.V(4).Infof("Attaching in container %s", containerId)
	container, err := r.containerStore.Get(containerId)
	if err != nil {
		return nil, fmt.Errorf("Container not found when execute sync command it")
	}
	remoteRuntime, err := r.remoteCRI.GetRemoteRuntime(container.PodSandbox.RuntimeHandler)
	if err != nil {
		return nil, err
	}
	if remoteRuntime != nil {
		return remoteRuntime.Attach(ctx, req)
	}
	if container.State != runtimeApi.ContainerState_CONTAINER_RUNNING {
		return nil, fmt.Errorf("Container is not started")
	}
	return r.adapter.Attach(container, req)
}

func (r *BabelFishRuntime) Exec(ctx context.Context, req *runtimeApi.ExecRequest) (*runtimeApi.ExecResponse, error) {
	containerId := req.ContainerId
	klog.V(4).Infof("Executing command in container %s", containerId)
	container, err := r.containerStore.Get(containerId)
	if err != nil {
		return nil, fmt.Errorf("Container not found when execute sync command it")
	}
	remoteRuntime, err := r.remoteCRI.GetRemoteRuntime(container.PodSandbox.RuntimeHandler)
	if err != nil {
		return nil, err
	}
	if remoteRuntime != nil {
		return remoteRuntime.Exec(ctx, req)
	}
	if container.State != runtimeApi.ContainerState_CONTAINER_RUNNING {
		return nil, fmt.Errorf("Container is not started")
	}
	return r.adapter.Exec(container, req)
}

func (r *BabelFishRuntime) ExecSync(ctx context.Context, req *runtimeApi.ExecSyncRequest) (*runtimeApi.ExecSyncResponse, error) {
	containerId := req.ContainerId
	klog.V(4).Infof("Executing synchronous command in container %s", containerId)
	container, err := r.containerStore.Get(containerId)
	if err != nil {
		return nil, fmt.Errorf("Container not found when execute sync command it")
	}
	remoteRuntime, err := r.remoteCRI.GetRemoteRuntime(container.PodSandbox.RuntimeHandler)
	if err != nil {
		return nil, err
	}
	if remoteRuntime != nil {
		r, er := remoteRuntime.ExecSync(ctx, req)
		return r, er
	}
	if container.State != runtimeApi.ContainerState_CONTAINER_RUNNING {
		return nil, fmt.Errorf("Container is not started")
	}
	return r.adapter.ExecSync(container, req.Cmd)
}
