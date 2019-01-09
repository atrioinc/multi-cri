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

package store

import (
	runtimeApi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

type ContainerMetadata struct {
	// ID is the container id.
	ID string
	// Name is the container name.
	Name string
	// Container state
	State runtimeApi.ContainerState
	// PodSandbox is the sandbox id the container belongs to.
	PodSandbox SandboxMetadata
	// config container properties.
	Config runtimeApi.ContainerConfig
	// image filesystem location
	Image *ImageMetadata
	//command
	Command []string
	//arguments
	Args []string
	// created time seconds
	CreatedAt int64
	// started time seconds
	StartedAt int64
	// finished time seconds
	FinishedAt int64
	// Container is a service, not run and finish
	IsService bool
	//Environment variables
	Environment map[string]string
	//Mount paths to bind
	Mounts []runtimeApi.Mount
	//Reason of state
	Reason string
	//Exit code
	ExitCode int
	//Port
	Port int
	// Process ID
	Pid int
	// Log file
	LogFile string
	// Extra information
	Extra map[string]string
}

type SandboxMetadata struct {
	// ID is the pod id.
	ID string
	// Container state
	State runtimeApi.PodSandboxState
	// List of pod containers
	Containers []string
	// config pod properties
	Config runtimeApi.PodSandboxConfig
	// Log file
	LogPath string
	// Network namespace
	NetNSPath string
	// created time seconds
	CreatedAt int64
	//Cgroups
	CgroupsParent string
	//Network IP
	IP string
	id int
	// ResourceManager variables
	VolumePath string
	//RuntimeClass
	RuntimeHandler string
}

type ImageMetadata struct {
	// ID is the pod id.
	ID string
	// Image name in repository.
	ImageName string
	// Local image path in the CRI node
	LocalPath string
	// Remote image path. URL image location
	RemotePath string
	//Tags
	RepoTags []string
	//Digest
	RepoDigest []string
	//Image size
	Size uint64
	// PodSandbox is the sandbox id the image belongs to.
	PodSandbox SandboxMetadata
	//Repo type
	RepoType RepoType
	//Repo
	Auth ImageAuth
}

type RepoType int

type ImageAuth runtimeApi.AuthConfig
