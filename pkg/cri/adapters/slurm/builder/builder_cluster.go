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

package builder

import (
	"multi-cri/pkg/cri/adapters/slurm/cmd"
	"multi-cri/pkg/cri/auth"
	"multi-cri/pkg/cri/store"
	"fmt"
	"path/filepath"
)

type ImageBuilderInCluster struct {
	MountPoint  string
	RemoteMount string
}

func NewImageBuilderInCluster(mountPoint string, remoteMount string) (ImageBuilder, error) {
	builder := ImageBuilderInCluster{MountPoint: mountPoint, RemoteMount: remoteMount}
	return builder, nil
}

func (builder ImageBuilderInCluster) PullImage(cm *store.ContainerMetadata) error {
	cm.Image.Size = 1 //It must to be set, otherwise k8s fails
	if builder.RemoteMount != "" {
		cm.Image.LocalPath = builder.RemoteMount
	}
	return nil
}

func (builder ImageBuilderInCluster) PullImageInCluster(cm *store.ContainerMetadata) error {
	cm.Image.LocalPath = getMountImagePath(cm, builder.MountPoint)

	if cm.Image.RepoType == store.LocalImageRepo {
		return PullLocalImage(cm)
	}

	authString := auth.ParseImageAuth(cm)
	imagePath := builder.GetImagePath(cm)
	command := fmt.Sprintf("%s singularity pull %s %s", authString, imagePath, cm.Image.RemotePath)
	client, err := cmd.CreateCMD(cm)
	if err != nil {
		return err
	}
	scriptPath := getRMImageScript(cm)

	if cm.Image.Size, err = client.PullImageScript(command, imagePath, scriptPath, cm.Environment); err != nil {
		return err
	}

	return nil
}

func (builder ImageBuilderInCluster) GetImagePath(cm *store.ContainerMetadata) string {
	return GetRMImagePath(cm, builder.MountPoint, builder.RemoteMount)
}

func GetRMImagePath(cm *store.ContainerMetadata, mountPoint, imageRemoteMount string) string {
	imageName := parseImageFileName(cm.Image.RemotePath)
	var folder string
	if imageRemoteMount != "" { // use image volume folder: <MOUNTHPATH>/.images/imageName
		folder = mountPoint
	} else { // use container volume parent folder: volName/.images/imageName
		volName := filepath.Base(cm.Extra["VolumePath"])
		folder = fmt.Sprintf("%s/%s", mountPoint, volName)
	}
	return fmt.Sprintf("$HOME/%s/.images/%s", folder, imageName)
}

func getRMImageScript(cm *store.ContainerMetadata) string {
	return fmt.Sprintf("%s/%s", cm.Extra["RMPath"], ImageScript)
}
