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

package adapters

import (
	"multi-cri/pkg/cri/store"

	runtimeApi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
	"k8s.io/kubernetes/pkg/kubelet/server/streaming"
)

const VolumeContainer = "/multicri"

type AdapterInterface interface {
	//Sandbox
	RunPodSandbox(sandbox *store.SandboxMetadata) error
	StopPodSandbox(sandbox *store.SandboxMetadata) error
	RemovePodSandbox(sandbox *store.SandboxMetadata) error
	PodSandboxStatus(sandbox *store.SandboxMetadata) error
	//CRI Version
	Version() (*runtimeApi.VersionResponse, error)
	//Container
	CreateContainer(cm *store.ContainerMetadata) error
	StartContainer(cm *store.ContainerMetadata) error
	StopContainer(cm *store.ContainerMetadata) error
	ContainerStatus(cm *store.ContainerMetadata) error
	ReopenContainerLog(cm *store.ContainerMetadata) error
	UpdateContainerResources(cm *store.ContainerMetadata) error
	//Pull Image
	PullImage(image *store.ImageMetadata) error
	ListImages(images []*runtimeApi.Image) error
	ImageStatus(image *store.ImageMetadata) error
	ImageFsInfo() (*runtimeApi.ImageFsInfoResponse, error)
	RemoveImage(image *store.ImageMetadata) error
	//Stream exec
	ExecSync(cm *store.ContainerMetadata, command []string) (*runtimeApi.ExecSyncResponse, error)
	Exec(cm *store.ContainerMetadata, req *runtimeApi.ExecRequest) (*runtimeApi.ExecResponse, error)
	Attach(cm *store.ContainerMetadata, req *runtimeApi.AttachRequest) (*runtimeApi.AttachResponse, error)
	NewStreamRuntime(c store.ContainerStoreInterface) streaming.Runtime
}
