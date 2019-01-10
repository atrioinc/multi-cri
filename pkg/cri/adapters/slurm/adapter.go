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

package slurm

import (
	"multi-cri/pkg/cri/adapters"
	"multi-cri/pkg/cri/adapters/slurm/builder"
	"multi-cri/pkg/cri/common"
	"fmt"

	runtimeApi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

const (
	SLURMADAPTERVERSION = "0.1.0"
	SLURMNAME           = "Adapter Slurm"
	MOUNTHPATH          = "multi-cri"
	StdoutFile          = "stdout.out"
	SterrFile           = "sterr.out"
	RunScript           = "run.sh"
)

type SlurmAdapter struct {
	MountPath        string
	Builder          builder.ImageBuilder
	ImageRemoteMount string
}

func NewSlurmAdapter() (adapters.AdapterInterface, error) {
	q := MOUNTHPATH
	b := false
	remoteDefault := ""
	imageRemoteMountPath := common.GetEnv("CRI_SLURM_IMAGE_REMOTE_MOUNT", &remoteDefault)
	mountP := common.GetEnv("CRI_SLURM_MOUNT_PATH", &q)

	var build builder.ImageBuilder
	var err error
	if common.GetBoolEnv("CRI_SLURM_BUILD_IN_CLUSTER", &b) {

		if build, err = builder.NewImageBuilderInCluster(mountP, imageRemoteMountPath); err != nil {
			return nil, err
		}
	} else {
		if build, err = builder.NewImageBuilderInCRI(imageRemoteMountPath); err != nil {
			return nil, err
		}
	}

	return SlurmAdapter{MountPath: mountP, Builder: build, ImageRemoteMount: imageRemoteMountPath}, nil
}

func (s SlurmAdapter) Version() (*runtimeApi.VersionResponse, error) {
	return &runtimeApi.VersionResponse{
		Version:           SLURMADAPTERVERSION,
		RuntimeName:       SLURMNAME,
		RuntimeVersion:    SLURMADAPTERVERSION,
		RuntimeApiVersion: SLURMADAPTERVERSION,
	}, nil
}

func getRMScriptPath(RMContainerPath string) string {
	return fmt.Sprintf("%s/%s", RMContainerPath, RunScript)
}

func getRMStderrPath(RMContainerPath string) string {
	return fmt.Sprintf("%s/%s", RMContainerPath, SterrFile)
}

func getRMStdoutPath(RMContainerPath string) string {
	return fmt.Sprintf("%s/%s", RMContainerPath, StdoutFile)
}
