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
	"golang.org/x/net/context"
	runtimeApi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

func (r RemoteCRIConfiguration) ReopenContainerLog(runtimeHandler string, ctx context.Context, req *runtimeApi.ReopenContainerLogRequest) (*runtimeApi.ReopenContainerLogResponse, error) {
	remoteRuntime, err := r.GetRemoteRuntime(runtimeHandler)
	if err != nil {
		return nil, err
	}
	if remoteRuntime != nil {
		return remoteRuntime.ReopenContainerLog(ctx, req)
	}
	return nil, nil
}

func (r RemoteCRIConfiguration) CreateContainer(runtimeHandler string, ctx context.Context, req *runtimeApi.CreateContainerRequest) (*runtimeApi.CreateContainerResponse, error) {
	remoteRuntime, err := r.GetRemoteRuntime(runtimeHandler)
	if err != nil {
		return nil, err
	}
	if remoteRuntime != nil {
		return remoteRuntime.CreateContainer(ctx, req)
	}
	return nil, nil
}

func (r RemoteCRIConfiguration) StartContainer(runtimeHandler string, ctx context.Context, req *runtimeApi.StartContainerRequest) (*runtimeApi.StartContainerResponse, error) {
	remoteRuntime, err := r.GetRemoteRuntime(runtimeHandler)
	if err != nil {
		return nil, err
	}
	if remoteRuntime != nil {
		return remoteRuntime.StartContainer(ctx, req)
	}
	return nil, nil
}

func (r RemoteCRIConfiguration) StopContainer(runtimeHandler string, ctx context.Context, req *runtimeApi.StopContainerRequest) (*runtimeApi.StopContainerResponse, error) {
	remoteRuntime, err := r.GetRemoteRuntime(runtimeHandler)
	if err != nil {
		return nil, err
	}
	if remoteRuntime != nil {
		return remoteRuntime.StopContainer(ctx, req)
	}
	return nil, nil
}

func (r RemoteCRIConfiguration) RemoveContainer(runtimeHandler string, ctx context.Context, req *runtimeApi.RemoveContainerRequest) (*runtimeApi.RemoveContainerResponse, error) {
	remoteRuntime, err := r.GetRemoteRuntime(runtimeHandler)
	if err != nil {
		return nil, err
	}
	if remoteRuntime != nil {
		return remoteRuntime.RemoveContainer(ctx, req)
	}
	return nil, nil
}

func (r RemoteCRIConfiguration) ContainerStatus(runtimeHandler string, ctx context.Context, req *runtimeApi.ContainerStatusRequest) (*runtimeApi.ContainerStatusResponse, error) {
	remoteRuntime, err := r.GetRemoteRuntime(runtimeHandler)
	if err != nil {
		return nil, err
	}
	if remoteRuntime != nil {
		return remoteRuntime.ContainerStatus(ctx, req)
	}
	return nil, nil
}

func (r RemoteCRIConfiguration) ContainerStats(runtimeHandler string, ctx context.Context, req *runtimeApi.ContainerStatsRequest) (*runtimeApi.ContainerStatsResponse, error) {
	remoteRuntime, err := r.GetRemoteRuntime(runtimeHandler)
	if err != nil {
		return nil, err
	}
	if remoteRuntime != nil {
		return remoteRuntime.ContainerStats(ctx, req)
	}
	return nil, nil
}

func (r RemoteCRIConfiguration) UpdateContainerResources(runtimeHandler string, ctx context.Context, req *runtimeApi.UpdateContainerResourcesRequest) (*runtimeApi.UpdateContainerResourcesResponse, error) {
	remoteRuntime, err := r.GetRemoteRuntime(runtimeHandler)
	if err != nil {
		return nil, err
	}
	if remoteRuntime != nil {
		return remoteRuntime.UpdateContainerResources(ctx, req)
	}
	return nil, nil
}

func (r RemoteCRIConfiguration) ListContainerStats(ctx context.Context, req *runtimeApi.ListContainerStatsRequest) ([]*runtimeApi.ContainerStats, error) {
	var remotePods []*runtimeApi.ContainerStats
	for _, remoteRuntime := range r.remoteCRIList {
		response, err := remoteRuntime.ListContainerStats(ctx, req)
		if err != nil {
			return nil, err
		}
		remotePods = append(remotePods, response.Stats...)
	}

	return remotePods, nil
}

func (r RemoteCRIConfiguration) ListContainers(ctx context.Context, req *runtimeApi.ListContainersRequest) ([]*runtimeApi.Container, error) {
	var remotePods []*runtimeApi.Container
	for _, remoteRuntime := range r.remoteCRIList {
		response, err := remoteRuntime.ListContainers(ctx, req)
		if err != nil {
			return nil, err
		}
		remotePods = append(remotePods, response.Containers...)
	}

	return remotePods, nil
}
