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
	"multi-cri/pkg/cri/store"
	"fmt"

	"time"

	runtimeApi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

func (s SlurmAdapter) PullImage(image *store.ImageMetadata) error {
	if image.RepoType == store.UnknownImageRepo {
		return fmt.Errorf("Image repository type not supported by multi-cri %s ", image.RemotePath)
	}
	container := &store.ContainerMetadata{Image: image}
	return s.Builder.PullImage(container)
}

func (s SlurmAdapter) ListImages(images []*runtimeApi.Image) error {
	return nil
}

func (s SlurmAdapter) ImageStatus(image *store.ImageMetadata) error {
	return nil
}

func (s SlurmAdapter) ImageFsInfo() (*runtimeApi.ImageFsInfoResponse, error) {
	//todo control it properly
	filesystems := []*runtimeApi.FilesystemUsage{
		{
			Timestamp: time.Now().UnixNano(),
			UsedBytes: &runtimeApi.UInt64Value{Value: uint64(0)},
			FsId: &runtimeApi.FilesystemIdentifier{
				Mountpoint: s.MountPath,
			},
		},
	}
	return &runtimeApi.ImageFsInfoResponse{ImageFilesystems: filesystems}, nil
}

func (s SlurmAdapter) RemoveImage(image *store.ImageMetadata) error {
	return nil
}
