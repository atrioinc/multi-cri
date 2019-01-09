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
	"time"

	"cri-babelfish/pkg/cri/store"

	"golang.org/x/net/context"
	"k8s.io/klog"
	runtimeApi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

func (r *BabelFishRuntime) ReopenContainerLog(ctx context.Context, req *runtimeApi.ReopenContainerLogRequest) (*runtimeApi.ReopenContainerLogResponse, error) {
	c, err := r.containerStore.Get(req.GetContainerId())
	if err != nil {
		return nil, fmt.Errorf("Container not found")
	}
	response, err := r.remoteCRI.ReopenContainerLog(c.PodSandbox.RuntimeHandler, ctx, req)
	if response != nil {
		return response, err
	}
	return &runtimeApi.ReopenContainerLogResponse{}, r.adapter.ReopenContainerLog(c)
}

func (r *BabelFishRuntime) CreateContainer(ctx context.Context, req *runtimeApi.CreateContainerRequest) (*runtimeApi.CreateContainerResponse, error) {
	var err error

	podSandboxID := req.PodSandboxId
	klog.V(4).Infof("Creating container in sandbox %s", podSandboxID)
	sandbox, err := r.sandboxStore.Get(podSandboxID)
	if err != nil {
		return nil, err
	}
	response, err := r.remoteCRI.CreateContainer(sandbox.RuntimeHandler, ctx, req)
	if err != nil {
		return nil, err
	}

	var container *store.ContainerMetadata

	if response == nil {
		name := req.GetConfig().GetMetadata().Name
		image, err := r.imageStore.GetByID(req.Config.Image.Image)
		if err != nil {
			return nil, fmt.Errorf("Image not found")
		}
		state := runtimeApi.ContainerState_CONTAINER_CREATED
		createdAt := int64(time.Now().UnixNano())
		envVars := make(map[string]string)
		if req.Config.Envs != nil {
			for _, env := range req.Config.Envs {
				envVars[env.Key] = env.Value
			}
		}
		port := 0
		//Container port
		if len(req.SandboxConfig.PortMappings) > 0 && req.SandboxConfig.PortMappings[0].ContainerPort > 0 {
			port = int(req.SandboxConfig.PortMappings[0].ContainerPort)
		}

		container = r.containerStore.CreateContainerMetadata(name,
			sandbox, state, createdAt, image, req.GetConfig().Command, req.GetConfig().Args,
			true, *req.GetConfig(), envVars, port, nil)

		if err := r.adapter.CreateContainer(container); err != nil {
			klog.V(4).Info(err)
			return nil, err
		}

		// It can be modified on the container creation
		r.imageStore.Update(container.Image)
		r.containerStore.Update(container)

	} else {
		container = r.containerStore.CreateContainerMetadata("",
			sandbox, 0, 0, nil, req.GetConfig().Command, req.GetConfig().Args,
			true, *req.GetConfig(), nil, 0, &response.ContainerId)
	}

	sandbox.Containers = append(sandbox.Containers, container.ID)
	r.sandboxStore.Update(sandbox)
	return &runtimeApi.CreateContainerResponse{ContainerId: container.ID}, err
}

func (r *BabelFishRuntime) StartContainer(ctx context.Context, req *runtimeApi.StartContainerRequest) (*runtimeApi.StartContainerResponse, error) {
	containerId := req.ContainerId
	klog.V(4).Infof("Starting container %s", containerId)
	cm, err := r.containerStore.Get(containerId)
	if err != nil {
		return nil, fmt.Errorf("Container not found when starting it")
	}

	response, err := r.remoteCRI.StartContainer(cm.PodSandbox.RuntimeHandler, ctx, req)

	if response != nil {
		return response, err
	}
	if cm.State == runtimeApi.ContainerState_CONTAINER_RUNNING {
		return &runtimeApi.StartContainerResponse{}, fmt.Errorf("Container already started")
	}
	if cm.State != runtimeApi.ContainerState_CONTAINER_CREATED {
		return &runtimeApi.StartContainerResponse{}, fmt.Errorf("Container failed")
	}

	if err = r.adapter.StartContainer(cm); err != nil {
		klog.V(4).Info(err)
		cm.State = runtimeApi.ContainerState_CONTAINER_EXITED
		cm.Reason = "Start container fails"
		cm.ExitCode = 1
		cm.FinishedAt = time.Now().Unix()
		response = nil
		cm.Reason = "ContainerCannotRun"
	} else {
		cm.State = runtimeApi.ContainerState_CONTAINER_RUNNING
		cm.StartedAt = int64(time.Now().UnixNano())
		cm.Reason = "Start container ok"
		response = &runtimeApi.StartContainerResponse{}
	}

	r.containerStore.Update(cm)

	return response, err
}

func (r *BabelFishRuntime) StopContainer(ctx context.Context, req *runtimeApi.StopContainerRequest) (*runtimeApi.StopContainerResponse, error) {
	containerId := req.ContainerId
	klog.V(4).Infof("Stopping container %s", containerId)
	cm, errGet := r.containerStore.Get(containerId)

	var runtimeClass string
	if errGet == nil {
		runtimeClass = cm.PodSandbox.RuntimeHandler
	}
	response, err := r.remoteCRI.StopContainer(runtimeClass, ctx, req)
	if response != nil {
		return response, err
	}

	if errGet != nil {
		return nil, errGet
	}

	if cm.State != runtimeApi.ContainerState_CONTAINER_RUNNING {
		return &runtimeApi.StopContainerResponse{}, nil
	}

	if err = r.adapter.StopContainer(cm); err != nil {
		klog.V(4).Info(err)
		return nil, err
	}

	cm.State = runtimeApi.ContainerState_CONTAINER_EXITED
	cm.FinishedAt = int64(time.Now().UnixNano())
	r.containerStore.Update(cm)

	return &runtimeApi.StopContainerResponse{}, nil
}

func (r *BabelFishRuntime) ListContainers(ctx context.Context, req *runtimeApi.ListContainersRequest) (*runtimeApi.ListContainersResponse, error) {
	klog.V(4).Infof("List containers")
	babelCRI := r.remoteCRI.BabelFishRuntimeName()
	containers := r.containerStore.ListK8s(req.Filter.Id, req.Filter.PodSandboxId, req.Filter.LabelSelector, req.Filter.State, babelCRI)
	remoteContainers, err := r.remoteCRI.ListContainers(ctx, req)
	if err != nil {
		return nil, err
	}
	containers = append(containers, remoteContainers...)
	return &runtimeApi.ListContainersResponse{Containers: containers}, nil
}

func (r *BabelFishRuntime) RemoveContainer(ctx context.Context, req *runtimeApi.RemoveContainerRequest) (*runtimeApi.RemoveContainerResponse, error) {
	containerId := req.ContainerId
	klog.V(4).Infof("Removing container %s", containerId)
	cm, errGet := r.containerStore.Get(containerId)
	var runtimeClass string
	if errGet == nil {
		runtimeClass = cm.PodSandbox.RuntimeHandler
	}
	response, err := r.remoteCRI.RemoveContainer(runtimeClass, ctx, req)
	if err != nil {
		return nil, err
	}
	if response == nil {
		if errGet != nil {
			return nil, errGet
		}
		if cm.State == runtimeApi.ContainerState_CONTAINER_RUNNING {
			return &runtimeApi.RemoveContainerResponse{}, fmt.Errorf("Running containers can not be deleted %s", containerId)
		}
		if err := r.adapter.ReopenContainerLog(cm); err != nil {
			klog.V(4).Info(err)
			return nil, err
		}
	}

	r.sandboxStore.RemoveContainer(cm.PodSandbox.ID, containerId)
	return &runtimeApi.RemoveContainerResponse{}, nil
}

func (r *BabelFishRuntime) ContainerStatus(ctx context.Context, req *runtimeApi.ContainerStatusRequest) (*runtimeApi.ContainerStatusResponse, error) {
	containerId := req.ContainerId
	klog.V(4).Infof("Getting status from container %s", containerId)
	cm, errGet := r.containerStore.Get(containerId)
	var runtimeClass string
	if errGet == nil {
		runtimeClass = cm.PodSandbox.RuntimeHandler
	}
	response, err := r.remoteCRI.ContainerStatus(runtimeClass, ctx, req)
	if err != nil {
		return nil, err
	}
	if response == nil {
		if cm.State != runtimeApi.ContainerState_CONTAINER_EXITED {
			if err = r.adapter.ContainerStatus(cm); err != nil {
				klog.V(4).Info(err)
				manageContainerError(cm, err)
			}
		}
		r.containerStore.Update(cm)
		containerStatus := store.GetK8sContainerStatus(cm)
		response = &runtimeApi.ContainerStatusResponse{Status: &containerStatus}
	}
	return response, nil
}

func (r *BabelFishRuntime) ListContainerStats(ctx context.Context, req *runtimeApi.ListContainerStatsRequest) (*runtimeApi.ListContainerStatsResponse, error) {
	klog.V(4).Info("List container stats")
	_, err := r.sandboxStore.Get(req.Filter.PodSandboxId)
	if err != nil {
		return nil, fmt.Errorf("Containers not found when listing them")
	}

	stats := []*runtimeApi.ContainerStats{}
	for _, K8Container := range r.containerStore.ListK8s(req.Filter.Id, req.Filter.GetPodSandboxId(), req.Filter.LabelSelector, nil, r.remoteCRI.BabelFishRuntimeName()) {
		attributes := runtimeApi.ContainerAttributes{K8Container.Id,
			K8Container.Metadata, K8Container.Labels,
			K8Container.Annotations,
		}
		stats = append(stats, &runtimeApi.ContainerStats{Attributes: &attributes})
	}
	remoteStats, err := r.remoteCRI.ListContainerStats(ctx, req)
	if err != nil {
		return nil, err
	}
	stats = append(stats, remoteStats...)
	return &runtimeApi.ListContainerStatsResponse{Stats: stats}, nil
}

func (r *BabelFishRuntime) ContainerStats(ctx context.Context, req *runtimeApi.ContainerStatsRequest) (*runtimeApi.ContainerStatsResponse, error) {
	containerId := req.ContainerId
	klog.V(4).Infof("Getting stats from container %s", containerId)
	container, err := r.containerStore.Get(containerId)
	if err != nil {
		container = &store.ContainerMetadata{}
	}
	response, err := r.remoteCRI.ContainerStats(container.PodSandbox.RuntimeHandler, ctx, req)
	if err != nil {
		return nil, err
	}

	if response == nil {
		K8Container := store.ParseToK8sContainer(container)
		attributes := runtimeApi.ContainerAttributes{K8Container.Id,
			K8Container.Metadata, K8Container.Labels,
			K8Container.Annotations,
		}
		stat := &runtimeApi.ContainerStats{Attributes: &attributes}
		response = &runtimeApi.ContainerStatsResponse{stat}
	}
	return response, nil

}

func (r *BabelFishRuntime) UpdateContainerResources(ctx context.Context, req *runtimeApi.UpdateContainerResourcesRequest) (*runtimeApi.UpdateContainerResourcesResponse, error) {
	klog.V(4).Infof("Updating container resources%s", req.GetContainerId())
	cm, err := r.containerStore.Get(req.GetContainerId())
	if err != nil {
		cm = &store.ContainerMetadata{}
	}
	response, err := r.remoteCRI.UpdateContainerResources(cm.PodSandbox.RuntimeHandler, ctx, req)
	if response != nil {
		response = &runtimeApi.UpdateContainerResourcesResponse{}
	}
	//TODO: implement it
	return response, r.adapter.UpdateContainerResources(cm)
}

func manageContainerError(cm *store.ContainerMetadata, err error) {
	cm.State = runtimeApi.ContainerState_CONTAINER_EXITED
	cm.ExitCode = 1
	cm.Reason = err.Error()
}
