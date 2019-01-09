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
	"cri-babelfish/pkg/cri/common/file"
	"cri-babelfish/pkg/cri/store"
	"fmt"
	"strings"
)

const (
	ImageScript = "image.sh"
)

type ImageBuilder interface {
	PullImage(cm *store.ContainerMetadata) error
	PullImageInCluster(cm *store.ContainerMetadata) error
	GetImagePath(cm *store.ContainerMetadata) string
}

func parseImageFileName(imageName string) string {
	chars := []string{":", "/"}
	for _, c := range chars {
		imageName = strings.Replace(imageName, c, ".", -1)
	}
	return imageName
}

// Setup local image path in pod
func getMountImagePath(cm *store.ContainerMetadata, imageRemoteMount string) string {
	imageName := parseImageFileName(cm.Image.RemotePath)
	var folder string
	if imageRemoteMount != "" {
		folder = cm.Image.LocalPath
	} else {
		folder = fmt.Sprintf("%s/.images", cm.Extra["VolumePath"])
	}
	if strings.HasSuffix(folder, imageName) {
		return folder
	}
	return fmt.Sprintf("%s/%s", folder, imageName)
}

func PullLocalImage(cm *store.ContainerMetadata) error {
	return file.CopyLocalImage(cm.Image.LocalPath, getLocalImagePath(cm))
}

func getLocalImagePath(cm *store.ContainerMetadata) string {
	return fmt.Sprintf("%s/%s", cm.Extra["VolumePath"], cleanLocalFileName(cm))
}

func cleanLocalFileName(cm *store.ContainerMetadata) string {
	return strings.Split(cm.Image.RemotePath, ":")[0] //Clean version setup by k8s
}
