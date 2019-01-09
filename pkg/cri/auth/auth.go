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

package auth

import (
	"cri-babelfish/pkg/cri/store"
	"fmt"

	"k8s.io/klog"
)

func authImageDocker(image *store.ImageMetadata) string {
	var cmd string
	if image.Auth.Username != "" && image.Auth.Password != "" {
		cmd = fmt.Sprintf("%s SINGULARITY_DOCKER_USERNAME=%s ", cmd, image.Auth.Username)
		cmd = fmt.Sprintf("%s SINGULARITY_DOCKER_PASSWORD=%s ", cmd, image.Auth.Password)
	}
	return cmd
}

func authImageSingularityHub(image *store.ImageMetadata) string {
	var cmd string
	klog.Warning("Authentication for Singularity hub is not implemented")
	return cmd
}

func ParseImageAuth(cm *store.ContainerMetadata) (auth string) {

	if cm.Image.RepoType == store.DockerRepositoryImageRepo {
		auth = authImageDocker(cm.Image)

	} else if cm.Image.RepoType == store.SingularityRepositoryImageRepo {
		auth = authImageSingularityHub(cm.Image)
	}
	return auth
}
