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

	"multi-cri/pkg/cri/store"

	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

func createContaier(container string, service CRIMulticriService) (string, error) {
	podName := FAKESANDBOXID
	podId, err := createPod(podName, service)
	if err != nil {
		return "", err
	}
	image := store.CRIDockerRepository + "server:v1"
	imageresponse, err := pullImage(image, service)
	if err != nil {
		return "", err
	}
	req := NewCreateContainerRequest(podId, container, imageresponse.ImageRef)
	response, err := service.CreateContainer(nil, &req)
	if err != nil {
		return "", err
	}
	return response.ContainerId, nil
}

//Test create and delete container
func TestUnitCreateDeleteContainer(t *testing.T) {
	service := NewFakeCRIService(false)
	name := "test1"
	podName := "testPod"
	podId, err := createPod(podName, service)
	if err != nil {
		t.Fatal(err)
	}
	image := FAKEIMAGE_DOCKER
	imageresponse, err := pullImage(image, service)
	if err != nil {
		t.Fatal(err)
	}
	containerReq := NewCreateContainerRequest(podId, name, imageresponse.ImageRef)

	container, err := service.CreateContainer(nil, &containerReq)
	if err != nil {
		t.Fatal("Create container fails:", err)
	}

	statusReq := NewContainerStatusRequest(container.ContainerId)
	outs, errs := service.ContainerStatus(nil, &statusReq)
	if errs != nil {
		t.Fatal("Can not retrieve container status:", errs)
	}
	status := outs.Status.State.String()
	if status != runtimeapi.ContainerState_CONTAINER_CREATED.String() {
		t.Fatal("Container should be created")
	}
	removeReq := NewContainerRemoveRequest(container.ContainerId)
	_, erre := service.RemoveContainer(nil, &removeReq)
	if erre != nil {
		t.Fatal("Can not delete container status:", errs)
	}
	outs, errs = service.ContainerStatus(nil, &statusReq)
	if errs != nil {
		t.Fatal("Container was not deleted", container.ContainerId)
	}
}

//Test create container has error if pod does no exits
func TestUnitCreateContainerErrorNoPod(t *testing.T) {
	name := "test2"
	pod := "noexists"
	containerReq := NewCreateContainerRequest(pod, name, FAKEIMAGE_DOCKER)
	service := NewFakeCRIService(false)
	_, err := service.CreateContainer(nil, &containerReq)
	if err == nil {
		t.Fatal("Create container should fails with pod: ", pod)
	}
}

//Test full container life cycle
func TestUnitCreateStartDeleteContainer(t *testing.T) {
	service := NewFakeCRIService(false)
	name := "test3"
	podName := "testPod"
	podId, err := createPod(podName, service)
	if err != nil {
		t.Fatal(err)
	}
	image := FAKEIMAGE_DOCKER
	imageresponse, err := pullImage(image, service)
	if err != nil {
		t.Fatal(err)
	}
	containerReq := NewCreateContainerRequest(podId, name, imageresponse.ImageRef)

	container, err := service.CreateContainer(nil, &containerReq)
	if err != nil {
		t.Fatal("Create container fails: ", err)
	}
	statusReq := NewContainerStatusRequest(container.ContainerId)
	outs, errs := service.ContainerStatus(nil, &statusReq)
	if errs != nil {
		t.Fatal("Can not retrieve container status: ", errs)
	}
	status := outs.Status.State.String()
	if status != runtimeapi.ContainerState_CONTAINER_CREATED.String() {
		t.Fatal("Container should be created")
	}
	startReq := NewContainerStartRequest(container.ContainerId)
	_, errstart := service.StartContainer(nil, &startReq)
	if errstart != nil {
		t.Fatal("Can not start container status: ", errstart)
	}
	outs, errs = service.ContainerStatus(nil, &statusReq)
	if errs != nil {
		t.Fatal("Can not retrieve container status: ", errs)
	}
	if outs.Status.State.String() != runtimeapi.ContainerState_CONTAINER_RUNNING.String() {
		t.Fatal("Container should be created")
	}
	stopReq := NewContainerStopRequest(container.ContainerId)
	_, errstop := service.StopContainer(nil, &stopReq)
	if errstop != nil {
		t.Fatal("Can not stop container: ", errstop)
	}
	outs, errs = service.ContainerStatus(nil, &statusReq)
	if errs != nil {
		t.Fatal("Can not retrieve container status:", errs)
	}
	if outs.Status.State.String() != runtimeapi.ContainerState_CONTAINER_EXITED.String() {
		t.Fatal("Container should be created")
	}
	removeReq := NewContainerRemoveRequest(container.ContainerId)
	_, erre := service.RemoveContainer(nil, &removeReq)
	if erre != nil {
		t.Fatal("Can not delete container status: ", erre)
	}
	outs, errs = service.ContainerStatus(nil, &statusReq)
	if errs != nil {
		t.Fatal("Container was not deleted: ", container.ContainerId)
	}
}

//Test can not delete container with running
func TestUnitCreateStartDeleteContainerErr(t *testing.T) {
	service := NewFakeCRIService(false)
	name := "test3"
	podName := "testPod"
	podId, err := createPod(podName, service)
	if err != nil {
		t.Fatal(err)
	}
	image := FAKEIMAGE_DOCKER
	imageresponse, err := pullImage(image, service)
	if err != nil {
		t.Fatal(err)
	}
	containerReq := NewCreateContainerRequest(podId, name, imageresponse.ImageRef)
	container, err := service.CreateContainer(nil, &containerReq)
	if err != nil {
		t.Fatal("Create container fails: ", err)
	}
	statusReq := NewContainerStatusRequest(container.ContainerId)
	outs, errs := service.ContainerStatus(nil, &statusReq)
	if errs != nil {
		t.Fatal("Can not retrieve container status: ", errs)
	}
	status := outs.Status.State.String()
	if status != runtimeapi.ContainerState_CONTAINER_CREATED.String() {
		t.Fatal("Container should be created")
	}
	startReq := NewContainerStartRequest(container.ContainerId)
	_, errstart := service.StartContainer(nil, &startReq)
	if errstart != nil {
		t.Fatal("Can not start container status: ", errstart)
	}
	removeReq := NewContainerRemoveRequest(container.ContainerId)
	_, erre := service.RemoveContainer(nil, &removeReq)
	if erre == nil {
		t.Fatal("Create container should fails with status running")
	}
}

//Test raise exception when try to stop non existing container
func TestUnitStopContainerErrorNoFound(t *testing.T) {
	containerId := "notfound"
	service := NewFakeCRIService(false)
	stopReq := NewContainerStopRequest(containerId)
	_, errstop := service.StopContainer(nil, &stopReq)
	if errstop == nil || !strings.Contains(errstop.Error(), "Container not found") {
		t.Fatal("Stop should raise not found exception. It is: ", errstop)
	}
}

//Test raise exception when try to start non existing container
func TestUnitStartContainerErrorNoFound(t *testing.T) {
	containerId := "notfound"
	service := NewFakeCRIService(false)
	startReq := NewContainerStartRequest(containerId)
	_, errstart := service.StartContainer(nil, &startReq)
	if errstart == nil && !strings.Contains(errstart.Error(), "Container not found") {
		t.Fatal("Start should raise not found exception")
	}
}

//Test raise exception when try to remove non existing container
func TestUnitRemoveContainerErrorNoFound(t *testing.T) {
	containerId := "notfound"
	service := NewFakeCRIService(false)
	req := NewContainerRemoveRequest(containerId)
	_, err := service.RemoveContainer(nil, &req)
	if err == nil || !strings.Contains(err.Error(), "Container not found") {
		t.Fatal("Remove should raise not found exception")
	}
}

// Test stop container manage command client error
func TestUnitStopContainerErrorClient(t *testing.T) {
	name := "erroclientStop"
	service_ok := NewFakeCRIService(true)
	containerId, err := createContaier(name, service_ok)
	if err != nil {
		t.Fatal(err)
	}
	containerReqStart := NewContainerStartRequest(containerId)
	_, err = service_ok.StartContainer(nil, &containerReqStart)
	if err != nil {
		t.Fatal("Start container fails: ", err)
	}
	//service_fails := NewFakeCRIService(true)
	stopReq := NewContainerStopRequest(containerId)
	_, errstop := service_ok.StopContainer(nil, &stopReq)
	if errstop == nil || !strings.Contains(errstop.Error(), "Adapter fails") {
		t.Fatal("Stop should raise a exception from client")
	}
}

// Test create container manage command client error
func TestUnitCreateContainerErrorNoFound(t *testing.T) {
	containerId := FAKECONTAINERID
	service := NewFakeCRIService(true)
	req := NewCreateContainerRequest(NOEXISTSANDBOXID, containerId, FAKEIMAGE_DOCKER)
	_, err := service.CreateContainer(nil, &req)
	if err == nil || !strings.Contains(err.Error(), "Pod not found") {
		t.Fatal("Remove should raise not found exception:", err)
	}
}

//Test list containers
func TestUnitListContainers(t *testing.T) {
	service := NewFakeCRIService(false)
	_, err := createContaier("testList", service)
	if err != nil {
		t.Fatal(err)
	}
	filter := runtimeapi.ContainerFilter{}
	req := NewListContainersRequest(filter)
	out, err := service.ListContainers(nil, &req)
	if err != nil {
		t.Fatal("Failed when Listing")
	}
	if len(out.Containers) == 0 {
		t.Fatal("List does not find any container")
	}
}

//Test list containers with filter
func TestUnitListContainersWithFilter(t *testing.T) {
	service := NewFakeCRIService(false)
	_, err := createContaier("testList", service)
	if err != nil {
		t.Fatal(err)
	}
	labels := map[string]string{"io.kubernetes.pod.uid": FAKESANDBOXID}
	filter := runtimeapi.ContainerFilter{LabelSelector: labels}
	req := NewListContainersRequest(filter)
	out, err := service.ListContainers(nil, &req)
	if err != nil {
		t.Fatal("Failed when Listing")
	}
	if len(out.Containers) == 0 {
		t.Fatal("List does not find any container")
	}
}

//Test list containers which filters make response empty
func TestUnitListContainersEmpty(t *testing.T) {
	service := NewFakeCRIService(false)
	labels := map[string]string{"io.kubernetes.pod.uid": "noexits"}
	filter := runtimeapi.ContainerFilter{LabelSelector: labels}
	req := NewListContainersRequest(filter)
	out, err := service.ListContainers(nil, &req)
	if err != nil {
		t.Fatal("Failed when Listing")
	}
	if len(out.Containers) != 0 {
		t.Fatal("List should return a empty list of containers")
	}
}

//Test Get container stat
func TestUnitContainerStats(t *testing.T) {
	service := NewFakeCRIService(false)
	containerId, err := createContaier("testStats", service)
	if err != nil {
		t.Fatal(err)
	}

	req := NewContainerStatsRequest(containerId)
	out, err := service.ContainerStats(nil, &req)
	if err != nil {
		t.Fatal("Failed when getting stats from ", containerId)
	}
	if out.Stats.Attributes.Id != containerId {
		t.Fatal("Failed stats with wrong container id ", containerId)
	}

}

//Test List container stats
func TestUnitContainerListStats(t *testing.T) {
	service := NewFakeCRIService(false)
	containerId, err := createContaier("testStats", service)
	if err != nil {
		t.Fatal(err)
	}

	labels := map[string]string{"io.kubernetes.pod.uid": FAKESANDBOXID}
	filter := runtimeapi.ContainerStatsFilter{PodSandboxId: FAKESANDBOXID, LabelSelector: labels}
	req := NewListContainerStatsRequest(filter)
	out, err := service.ListContainerStats(nil, &req)
	if err != nil {
		t.Fatal("Failed when getting stats from .", containerId, err)
		return
	}
	if len(out.Stats) == 0 {
		t.Fatal("Failed listing stats of pod ", FAKESANDBOXID)
		return
	}

}
