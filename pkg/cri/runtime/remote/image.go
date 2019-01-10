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
	"strings"

	"golang.org/x/net/context"
	runtimeApi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

func (r RemoteCRIConfiguration) PullImage(ctx context.Context, req *runtimeApi.PullImageRequest) (res *runtimeApi.PullImageResponse, err error) {
	var remoteRuntime *RemoteCRIObject
	remoteRuntime, req.Image.Image = r.getImageManager(req.Image.Image)
	if remoteRuntime != nil {
		return remoteRuntime.PullImage(ctx, req)
	}
	return nil, nil
}

func (r RemoteCRIConfiguration) ImageStatus(ctx context.Context, req *runtimeApi.ImageStatusRequest) (*runtimeApi.ImageStatusResponse, error) {
	var remoteRuntime *RemoteCRIObject
	remoteRuntime, req.Image.Image = r.getImageManager(req.Image.Image)
	if remoteRuntime != nil {
		return remoteRuntime.ImageStatus(ctx, req)
	}
	return nil, nil
}

func (r RemoteCRIConfiguration) ListImages(ctx context.Context, req *runtimeApi.ListImagesRequest) (res []*runtimeApi.Image, err error) {
	var remotePods []*runtimeApi.Image
	for _, remoteRuntime := range r.remoteCRIList {
		response, err := remoteRuntime.ListImages(ctx, req)
		if err != nil {
			return nil, err
		}
		remotePods = append(remotePods, response.Images...)
	}

	return remotePods, nil
}

func (r RemoteCRIConfiguration) getImageManager(image string) (*RemoteCRIObject, string) {
	imageMulticri := MulticriRuntimeHandler + "/"
	if strings.HasPrefix(image, imageMulticri) {
		return nil, strings.TrimPrefix(image, imageMulticri)
	}
	for k, v := range r.remoteCRIList {
		if strings.HasPrefix(image, k) {
			return v, strings.TrimPrefix(image, k)
		}
	}
	if remoteRuntime, ok := r.remoteCRIList["default"]; ok {
		return remoteRuntime, image
	}
	return nil, image
}

func (r RemoteCRIConfiguration) RemoveImage(ctx context.Context, req *runtimeApi.RemoveImageRequest) (_ *runtimeApi.RemoveImageResponse, err error) {
	var remoteRuntime *RemoteCRIObject
	remoteRuntime, req.Image.Image = r.getImageManager(req.Image.Image)
	if remoteRuntime != nil {
		return remoteRuntime.RemoveImage(ctx, req)
	}
	return nil, nil
}
