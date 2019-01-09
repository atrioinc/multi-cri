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
	"strings"
	"testing"

	"cri-babelfish/pkg/cri/store"

	"cri-babelfish/pkg/cri/common/file"

	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

//Test full def file image cycle
func TestUnitPullImageDefFile(t *testing.T) {
	service := NewFakeCRIService(false)
	image := store.CRISLocalDefinitionFile + "serve2:v1"
	pod := FAKESANDBOXID
	labels := map[string]string{}
	req := NewPullImageRequest(image, pod, labels)
	out, err := service.PullImage(nil, &req)
	if err != nil {
		t.Fatal("Pull image virgo fails")
	}
	reqImageStatus := NewImageStatusRequest(image)
	imageStatus, err := service.ImageStatus(nil, &reqImageStatus)
	if err != nil {
		t.Fatal("Image status virgo fails")
	}
	if out.ImageRef != imageStatus.Image.Id {
		t.Fatal("Image ref wrong")
	}
	reqDelete := NewRemoveImageRequest(imageStatus.Image.Id)
	_, errRe := service.RemoveImage(nil, &reqDelete)
	if errRe != nil {
		t.Fatal("Delete image virgo fails")
	}
	reqStatus := NewImageStatusRequest(image)
	outStatus, err := service.ImageStatus(nil, &reqStatus)
	if err != nil {
		t.Fatal("Status image fails")
	}
	if outStatus.Image != nil {
		t.Fatal("Test should return empty output")
	}
}

//Test full local image cycle
func TestUnitPullImageLocal(t *testing.T) {
	service := NewFakeCRIService(false)
	image := store.CRISLocalRepository + "serve1:v1"
	//Mocking downloaVirgoContainer function
	file.CopyLocalImage = downloadFileFake
	pod := FAKESANDBOXID
	labels := map[string]string{}
	req := NewPullImageRequest(image, pod, labels)
	out, err := service.PullImage(nil, &req)
	if err != nil {
		t.Fatal("Pull image virgo fails")
	}
	if !strings.Contains(image, out.ImageRef) {
		t.Fatal("Image ref wrong")
	}
	reqDelete := NewRemoveImageRequest(image)
	_, errRe := service.RemoveImage(nil, &reqDelete)
	if errRe != nil {
		t.Fatal("Delete image virgo fails")
	}
	reqStatus := NewImageStatusRequest(image)
	outStatus, err := service.ImageStatus(nil, &reqStatus)
	if err != nil {
		t.Fatal("Status image fails")
	}
	if outStatus.Image != nil {
		t.Fatal("Test should return empty output")
	}
}

func pullImage(image string, service CRIBabelFishService) (*runtimeapi.PullImageResponse, error) {

	pod := FAKESANDBOXID
	req := NewPullImageRequest(image, pod, map[string]string{})
	return service.PullImage(nil, &req)

}

//Test image list
func TestUnitListImages(t *testing.T) {
	service := NewFakeCRIService(false)
	imageName := store.CRIDockerRepository + "serve1:v1"
	_, err := pullImage(imageName, service)
	if err != nil {
		t.Fatal("Pull image virgo fails")
	}
	filter := runtimeapi.ImageFilter{}
	reqList := NewListImagesRequest(filter)
	outList, err := service.ListImages(nil, &reqList)
	if err != nil {
		t.Fatal("List image fails")
	}
	if len(outList.Images) == 0 {
		t.Fatal("List image fails")
	}
	if outList.Images[0] == nil {
		t.Fatal("List image fails")
	}
}

//Test image list
func TestUnitListImagesFilter(t *testing.T) {
	service := NewFakeCRIService(false)
	imageName := FAKEIMAGE_LOCAL
	_, err := pullImage(imageName, service)
	if err != nil {
		t.Fatal("Pull image virgo fails")
	}
	filter := runtimeapi.ImageFilter{&runtimeapi.ImageSpec{imageName}}
	req := NewListImagesRequest(filter)
	out, err := service.ListImages(nil, &req)
	if err != nil {
		t.Fatal("List image raises error")
	}
	if len(out.Images) != 1 {
		t.Fatal("List image fails size")
	}
	if out.Images[0] == nil {
		t.Fatal("List image fails when check content")
	}
}

//Test image status def
func TestUnitImageStatusDockerRepository(t *testing.T) {
	service := NewFakeCRIService(false)
	image := store.CRIDockerRepository + "serve1:v1"
	_, err := pullImage(image, service)
	if err != nil {
		t.Fatal("Pull image virgo fails")
	}
	req := NewImageStatusRequest(image)
	out, err := service.ImageStatus(nil, &req)
	if err != nil {
		t.Fatal("Status image fails")
	}
	if len(out.Image.RepoDigests) == 0 || !strings.Contains(out.Image.RepoDigests[0], store.CRIDockerRepository) {
		t.Errorf("Repository image key fails")
	}
}

//Test image status singulariy-hub
func TestUnitImageStatusSingularityHub(t *testing.T) {
	service := NewFakeCRIService(false)
	image := store.CRISingularityHubRepository + "serve1:v1"
	_, err := pullImage(image, service)
	if err != nil {
		t.Fatal("Pull image virgo fails")
	}
	req := NewImageStatusRequest(image)
	out, err := service.ImageStatus(nil, &req)
	if err != nil {
		t.Errorf("List image fails")
	}
	if len(out.Image.RepoDigests) == 0 || !strings.Contains(out.Image.RepoDigests[0], store.CRISingularityHubRepository) {
		t.Errorf("Repository image key fails")
	}
}

//Test image status def file
func TestUnitImageStatusLocal(t *testing.T) {
	service := NewFakeCRIService(false)
	image := store.CRISLocalRepository + "serve1:v1"
	_, err := pullImage(image, service)
	if err != nil {
		t.Fatal("Pull image virgo fails")
	}
	req := NewImageStatusRequest(image)
	out, err := service.ImageStatus(nil, &req)
	if err != nil {
		t.Errorf("List image fails")
	}
	if len(out.Image.RepoDigests) == 0 || !strings.Contains(out.Image.RepoDigests[0], store.CRISLocalRepository) {
		t.Errorf("Repository image key fails")
	}
}

//Test image status not found
func TestUnitImageStatusNotFound(t *testing.T) {
	service := NewFakeCRIService(false)
	image := store.CRISingularityHubRepository + "noexist:v1"
	req := NewImageStatusRequest(image)
	out, err := service.ImageStatus(nil, &req)
	if err != nil {
		t.Errorf("Test should return err nil")
	}
	if out.Image != nil {
		t.Errorf("Test should return empty output")
	}
}

//Test ImageFsInfo. Not implemented
func TestUnitImageFsInfoNotImplemented(t *testing.T) {
	req := NewImageFsInfoRequest()
	service := NewFakeCRIService(false)
	out, err := service.ImageFsInfo(nil, &req)
	if !strings.Contains(err.Error(), "ImageFsInfo still not implemented") {
		t.Errorf("ImageFsInfo fails")
	}
	if out != nil {
		t.Errorf("Test should return empty output")
	}
}

//Test image status not found
func TestUnitRemoveImageNotFound(t *testing.T) {
	service := NewFakeCRIService(false)
	image := store.CRISingularityHubRepository + "noexist:v1"
	req := NewRemoveImageRequest(image)
	_, err := service.RemoveImage(nil, &req)
	if !strings.Contains(err.Error(), "Image not found") {
		t.Errorf("Test should raise Image not found")
	}
}
