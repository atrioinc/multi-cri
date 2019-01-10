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

	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

func createPod(podName string, service CRIMulticriService) (string, error) {
	req := NewCreateSandboxRequest(podName, "")
	response, err := service.RunPodSandbox(nil, &req)
	if err != nil {
		return "", nil
	}
	return response.PodSandboxId, nil
}

//Test full sandbox cycle
func TestUnitRunSanboxFull(t *testing.T) {
	pod := FAKESANDBOXID
	req := NewCreateSandboxRequest(pod, "")
	service := NewFakeCRIService(false)
	outRun, err := service.RunPodSandbox(nil, &req)
	if err != nil || outRun.PodSandboxId != pod {
		t.Errorf("Create container should fails with pod %s", pod)
	}
	reqStatus := NewStatusSandboxRequest(pod)
	outStatus, err := service.PodSandboxStatus(nil, &reqStatus)
	if err != nil || outStatus.Status.Id != pod {
		t.Errorf("Container status fails with pod %s", pod)
	}
	if outStatus.Status.State != runtimeapi.PodSandboxState_SANDBOX_READY {
		t.Errorf("Status should be READY")
	}
	reqStop := NewStopSandboxRequest(pod)
	outStop, err := service.StopPodSandbox(nil, &reqStop)
	if err != nil || outStop == nil {
		t.Errorf("Stop Fails")
	}
	outStatus, err = service.PodSandboxStatus(nil, &reqStatus)
	if err != nil || outStatus.Status.Id != pod {
		t.Errorf("Container status fails with pod %s", pod)
	}
	if outStatus.Status.State != runtimeapi.PodSandboxState_SANDBOX_NOTREADY {
		t.Errorf("Status should be NOT READY")
	}
	reqRemove := NewRemoveSandboxRequest(pod)
	outRemove, err := service.RemovePodSandbox(nil, &reqRemove)
	if err != nil || outRemove == nil {
		t.Errorf("Remove Fails")
	}
	outStatus, err = service.PodSandboxStatus(nil, &reqStatus)
	if err == nil && !strings.Contains(err.Error(), "Pod not found") {
		t.Errorf("Pod should not exists anymore: %s", pod)
	}
}

//Test run sandbox and try to remove when it is running
func TestUnitRunSanboxFailsRemovingRunning(t *testing.T) {
	pod := "podtest1"
	req := NewCreateSandboxRequest(pod, "")
	service := NewFakeCRIService(false)
	outRun, err := service.RunPodSandbox(nil, &req)
	if err != nil || outRun.PodSandboxId != pod {
		t.Errorf("Create container should fails with pod %s", pod)
	}
	reqStatus := NewStatusSandboxRequest(pod)
	outStatus, err := service.PodSandboxStatus(nil, &reqStatus)
	if err != nil || outStatus.Status.Id != pod {
		t.Errorf("Container status fails with pod %s", pod)
	}
	if outStatus.Status.State != runtimeapi.PodSandboxState_SANDBOX_READY {
		t.Errorf("Status should be READY")
	}
	reqRemove := NewRemoveSandboxRequest(pod)
	_, err = service.RemovePodSandbox(nil, &reqRemove)
	if err != nil && !strings.Contains(err.Error(), "is running") {
		t.Errorf("Pod should not be removed when running: %s", pod)
	}
}

//Test list pod sandbox no filter
func TestUnitListSandboxPods(t *testing.T) {
	service := NewFakeCRIService(false)
	podNew := "podtest1"
	reqNew := NewCreateSandboxRequest(podNew, "")
	_, err := service.RunPodSandbox(nil, &reqNew)
	if err != nil {
		t.Errorf("Create container should fails with pod %s", podNew)
	}
	filter := runtimeapi.PodSandboxFilter{}
	req := NewListSandboxRequest(filter)
	out, err := service.ListPodSandbox(nil, &req)
	if err != nil || len(out.Items) < 1 {
		t.Errorf("List error")
	}
}

//Test list pod sandbox filter not found
func TestUnitListSandboxPodsFilterNotFond(t *testing.T) {
	service := NewFakeCRIService(false)
	podNew := "podtest2"
	reqNew := NewCreateSandboxRequest(podNew, "")
	_, err := service.RunPodSandbox(nil, &reqNew)
	if err != nil {
		t.Errorf("Create container should fails with pod %s", podNew)
	}
	pod := "notfound"
	labelFilter := map[string]string{"io.kubernetes.pod.uid": pod}
	filter := runtimeapi.PodSandboxFilter{Id: pod, LabelSelector: labelFilter}
	req := NewListSandboxRequest(filter)
	out, err := service.ListPodSandbox(nil, &req)
	if err != nil || len(out.Items) != 0 {
		t.Errorf("List error")
	}
}

//Test list pod sandbox with filters
func TestUnitListSandboxPodsFilterLabel(t *testing.T) {
	service := NewFakeCRIService(false)
	podNew := "podtest3"
	reqNew := NewCreateSandboxRequest(podNew, "")
	podResponse, err := service.RunPodSandbox(nil, &reqNew)
	if err != nil {
		t.Errorf("Create container shouldn't fail with pod %s", podNew)
	}
	labelFilter := map[string]string{"io.kubernetes.pod.uid": podResponse.PodSandboxId}
	filter := runtimeapi.PodSandboxFilter{Id: "", LabelSelector: labelFilter}
	req := NewListSandboxRequest(filter)
	out, err := service.ListPodSandbox(nil, &req)
	if err != nil || len(out.Items) != 1 {
		t.Errorf("List error")
	}
}

//Test portforwarding in pod sandbox. Not implemented
func TestUnitPortForwardingSanbox(t *testing.T) {
	port := []int32{1, 2}
	service := NewFakeCRIService(false)
	podId, err := createPod(FAKESANDBOXID, service)
	if err != nil {
		t.Fatal(err)
	}
	req := NewPortForwardRequest(podId, port)
	_, err = service.PortForward(nil, &req)
	if err != nil {
		t.Fatal("Portforward fails")
	}
}

//Test remove pod sandbox singularity client error
func TestUnitRemoveSanboxCliError(t *testing.T) {
	pod := "podtestRemoveError"
	req := NewRemoveSandboxRequest(pod)
	service := NewFakeCRIService(true)
	_, err := service.RemovePodSandbox(nil, &req)
	if err == nil && !strings.Contains(err.Error(), "Fail client") {
		t.Errorf("Test should fail because singularity client fails")
	}
}

//Test stop pod sandbox singularity client error
func TestUnitStopSanboxError(t *testing.T) {
	pod := "podtestStopError"
	req := NewStopSandboxRequest(pod)
	service := NewFakeCRIService(false)
	_, err := service.StopPodSandbox(nil, &req)
	if err == nil || !strings.Contains(err.Error(), "Sandbox not found when stopping it") {
		t.Errorf("Test should fail because there is not singularity tag:%s ", err)
	}
}

//Test remove pod sandbox Error not found
func TestUnitRemoveSanboxError(t *testing.T) {
	pod := "podtestRemoveError"
	req := NewRemoveSandboxRequest(pod)
	service := NewFakeCRIService(false)
	_, err := service.RemovePodSandbox(nil, &req)
	if err == nil && !strings.Contains(err.Error(), "Sandbox not found when stopping it") {
		t.Errorf("Test should fail because there is not singularity tag %s", err)
	}
}

//Test status pod sandbox Error not found
func TestUnitStatusSanboxError(t *testing.T) {
	pod := "podtestStatusError"
	req := NewStatusSandboxRequest(pod)
	service := NewFakeCRIService(false)
	_, err := service.PodSandboxStatus(nil, &req)
	if err == nil && !strings.Contains(err.Error(), "Sandbox not found when stopping it") {
		t.Errorf("Test should fail because there is not singularity tag %s:", err)
	}
}

//Test status pod sandbox Error not found
func TestUnitPortForwardingSanboxError(t *testing.T) {
	service := NewFakeCRIService(false)
	podId, err := createPod(FAKESANDBOXID, service)
	if err != nil {
		t.Fatal(err)
	}
	port := []int32{1, 2}
	req := NewPortForwardRequest(podId, port)

	_, err = service.PortForward(nil, &req)
	if err != nil {
		t.Fatal("Test was wrong")
	}
}
