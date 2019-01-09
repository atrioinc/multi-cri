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

func (r RemoteCRIConfiguration) RunPodSandbox(ctx context.Context, req *runtimeApi.RunPodSandboxRequest) (_ *runtimeApi.RunPodSandboxResponse, retErr error) {
	remoteRuntime, err := r.GetRemoteRuntime(req.GetRuntimeHandler())
	if err != nil {
		return nil, err
	}
	if remoteRuntime != nil {
		return remoteRuntime.RunPodSandbox(ctx, req)
	}
	return nil, nil
}

func (r RemoteCRIConfiguration) StopPodSandbox(runtimeHandler string, ctx context.Context, req *runtimeApi.StopPodSandboxRequest) (*runtimeApi.StopPodSandboxResponse, error) {
	remoteRuntime, err := r.GetRemoteRuntime(runtimeHandler)
	if err != nil {
		return nil, err
	}
	if remoteRuntime != nil {
		return remoteRuntime.StopPodSandbox(ctx, req)
	}
	return nil, nil
}

func (r RemoteCRIConfiguration) RemovePodSandbox(runtimeHandler string, ctx context.Context, req *runtimeApi.RemovePodSandboxRequest) (*runtimeApi.RemovePodSandboxResponse, error) {
	remoteRuntime, err := r.GetRemoteRuntime(runtimeHandler)
	if err != nil {
		return nil, err
	}
	if remoteRuntime != nil {
		return remoteRuntime.RemovePodSandbox(ctx, req)
	}
	return nil, nil
}

func (r RemoteCRIConfiguration) PodSandboxStatus(runtimeHandler string, ctx context.Context, req *runtimeApi.PodSandboxStatusRequest) (*runtimeApi.PodSandboxStatusResponse, error) {
	remoteRuntime, err := r.GetRemoteRuntime(runtimeHandler)
	if err != nil {
		return nil, err
	}
	if remoteRuntime != nil {
		return remoteRuntime.PodSandboxStatus(ctx, req)
	}
	return nil, nil
}

func (r RemoteCRIConfiguration) PortForward(runtimeHandler string, ctx context.Context, req *runtimeApi.PortForwardRequest) (*runtimeApi.PortForwardResponse, error) {
	remoteRuntime, err := r.GetRemoteRuntime(runtimeHandler)
	if err != nil {
		return nil, err
	}
	if remoteRuntime != nil {
		return remoteRuntime.PortForward(ctx, req)
	}
	return nil, nil
}

func (r RemoteCRIConfiguration) ListPodSandbox(ctx context.Context, req *runtimeApi.ListPodSandboxRequest) ([]*runtimeApi.PodSandbox, error) {
	var remotePods []*runtimeApi.PodSandbox
	for _, remoteRuntime := range r.remoteCRIList {
		response, err := remoteRuntime.ListPodSandbox(ctx, req)
		if err != nil {
			return nil, err
		}
		remotePods = append(remotePods, response.Items...)
	}

	return remotePods, nil
}
