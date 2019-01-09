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
	"cri-babelfish/pkg/cri/auth"
	"cri-babelfish/pkg/cri/common/cmd"
	"cri-babelfish/pkg/cri/common/file"
	"cri-babelfish/pkg/cri/store"
	"fmt"
	"path"
	"strings"
)

type ImageBuilderInCRI struct {
	client      cmd.SingularityCLI
	RemoteMount string
}

func NewImageBuilderInCRI(remoteMount string) (ImageBuilder, error) {
	c, err := cmd.NewSingularityCLI("", cmd.CLIConfig{})
	if err != nil {
		return nil, err
	}

	builder := ImageBuilderInCRI{
		client:      c,
		RemoteMount: remoteMount,
	}
	return builder, nil
}

func (builder ImageBuilderInCRI) PullImage(cm *store.ContainerMetadata) error {
	if cm.Image.RepoType == store.LocalImageRepo {
		cm.Image.LocalPath = getMountImagePath(cm, builder.RemoteMount)
		return nil
	}
	var size int64
	var err error
	authString := auth.ParseImageAuth(cm)

	cm.Image.LocalPath = builder.GetImagePath(cm)
	if err := file.EnsurePathExist(path.Dir(cm.Image.LocalPath)); err != nil {
		return err
	}
	if err := builder.client.SingularityPullImage(cm.Image.LocalPath, cm.Image.RemotePath, authString); err != nil {
		return err
	}
	if size, err = file.FileSize(cm.Image.LocalPath); err != nil {
		return err
	}
	cm.Image.Size = uint64(size)
	return nil
}

func (builder ImageBuilderInCRI) PullImageInCluster(cm *store.ContainerMetadata) error {
	if cm.Image.RepoType == store.LocalImageRepo {
		return PullLocalImage(cm)
	}
	if builder.RemoteMount == "" {
		//copy to the container volume parent folder: /volumeName/.images/ximage
		volumePath := getMountImagePath(cm, builder.RemoteMount)
		if err := file.EnsurePathExist(path.Dir(volumePath)); err != nil {
			return err
		}
		//todo check if it exists before copying it
		return file.CopyLocalImage(volumePath, cm.Image.LocalPath)
	}

	return nil
}

func (builder ImageBuilderInCRI) GetImagePath(cm *store.ContainerMetadata) string {
	imageName := parseImageFileName(cm.Image.RemotePath)
	var folder string
	if builder.RemoteMount != "" { // use image volume folder: <CRI_SLURM_IMAGE_REMOTE_MOUNT>/imageName
		folder = builder.RemoteMount
	} else { //use cri node local image cache: <resources-cache-path>/.images/
		folder = cm.Image.LocalPath
	}
	if strings.HasSuffix(folder, imageName) {
		return folder
	}
	return fmt.Sprintf("%s/%s", folder, imageName)
}
