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

	"cri-babelfish/pkg/cri/store"

	"golang.org/x/net/context"
	"k8s.io/klog"
	runtimeApi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

func (r *BabelFishRuntime) PullImage(ctx context.Context, req *runtimeApi.PullImageRequest) (res *runtimeApi.PullImageResponse, err error) {
	klog.V(4).Infof("Pulling image... %s", req.Image.Image)

	response, err := r.remoteCRI.PullImage(ctx, req)
	if response != nil {
		return response, err
	}
	if err != nil {
		return nil, err
	}

	imageMetadata, err := r.imageStore.CreateImageMetadata(req)
	if err != nil {
		return nil, err
	}

	if err := store.ParseImage(imageMetadata); err != nil {
		return nil, err
	}

	if err = r.adapter.PullImage(imageMetadata); err != nil {
		r.imageStore.Remove(imageMetadata.ID)
		return &runtimeApi.PullImageResponse{}, err
	}

	r.imageStore.Update(imageMetadata)
	klog.V(4).Infof("Image %s successfully pulled", req.Image.Image)
	return &runtimeApi.PullImageResponse{ImageRef: imageMetadata.RemotePath}, nil
}

func (r *BabelFishRuntime) ListImages(ctx context.Context, req *runtimeApi.ListImagesRequest) (res *runtimeApi.ListImagesResponse, err error) {
	klog.V(4).Infof("List images")
	images := r.imageStore.ListK8s(req.Filter)

	remoteList, err := r.remoteCRI.ListImages(ctx, req)
	images = append(images, remoteList...)

	if err := r.adapter.ListImages(images); err != nil {
		return nil, err
	}

	return &runtimeApi.ListImagesResponse{Images: images}, nil
}

func (r *BabelFishRuntime) ImageStatus(ctx context.Context, req *runtimeApi.ImageStatusRequest) (*runtimeApi.ImageStatusResponse, error) {
	klog.V(4).Infof("Getting image status... %s", req.Image.Image)

	response, err := r.remoteCRI.ImageStatus(ctx, req)
	if response != nil {
		return response, err
	}
	if err != nil {
		return nil, err
	}

	image, err := r.imageStore.GetByID(req.Image.Image)
	if err != nil {
		return &runtimeApi.ImageStatusResponse{}, nil
	}
	if err := r.adapter.ImageStatus(image); err != nil {
		return nil, err
	}
	imageStatus := store.ParseToK8sImage(image)
	return &runtimeApi.ImageStatusResponse{Image: imageStatus}, nil
}

func (r *BabelFishRuntime) RemoveImage(ctx context.Context, req *runtimeApi.RemoveImageRequest) (_ *runtimeApi.RemoveImageResponse, err error) {
	klog.V(4).Infof("Removing image... %s", req.Image.Image)
	response, err := r.remoteCRI.RemoveImage(ctx, req)
	if response != nil {
		return response, err
	}
	if err != nil {
		return nil, err
	}

	image, err := r.imageStore.GetByID(req.Image.Image)
	if err != nil {
		return nil, fmt.Errorf("Image not found")
	}

	if err := r.adapter.RemoveImage(image); err != nil {
		return nil, err
	}
	r.imageStore.Remove(image.ID)
	klog.V(4).Infof("Image %s successfully removed", req.Image.Image)
	return &runtimeApi.RemoveImageResponse{}, nil
}

func (r *BabelFishRuntime) ImageFsInfo(ctx context.Context, req *runtimeApi.ImageFsInfoRequest) (*runtimeApi.ImageFsInfoResponse, error) {
	klog.V(4).Info("ImageFsInfo")
	remoteRuntime, err := r.remoteCRI.GetRemoteRuntime("default")
	if err != nil {
		return nil, err
	}
	if remoteRuntime != nil {
		return remoteRuntime.ImageFsInfo(ctx, req)
	}
	return r.adapter.ImageFsInfo()
}
