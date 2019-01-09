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
	"testing"
)

func TestUnitAttach(t *testing.T) {
	service := NewFakeCRIService(false)
	containerId, err := createContaier(FAKECONTAINERID, service)
	if err != nil {
		t.Fatal(err)
	}
	reqStart := NewContainerStartRequest(containerId)
	_, err = service.StartContainer(nil, &reqStart)
	if err != nil {
		t.Fatal(err)
	}
	req := NewAttachRequest(containerId)
	_, err = service.Attach(nil, &req)
	if err != nil {
		t.Fatal("Attach fail ", err)
	}
}

func TestUnitExecContainerNotRunning(t *testing.T) {
	command := []string{"echo", "hola"}
	service := NewFakeCRIService(true)
	containerId, err := createContaier(FAKECONTAINERID, service)
	if err != nil {
		t.Fatal(err)
	}
	req := NewExecRequest(containerId, command)
	_, err = service.Exec(nil, &req)
	if err == nil {
		t.Errorf("Exec shouldn fail. %s", err)
	}
}

func TestUnitExecContainer(t *testing.T) {
	command := []string{"echo", "hola"}
	service := NewFakeCRIService(true)
	containerId, err := createContaier(FAKECONTAINERID, service)
	if err != nil {
		t.Fatal(err)
	}
	reqStart := NewContainerStartRequest(containerId)
	_, err = service.StartContainer(nil, &reqStart)
	if err != nil {
		t.Fatal(err)
	}
	req := NewExecRequest(containerId, command)
	_, err = service.Exec(nil, &req)
	if err != nil {
		t.Errorf("Exec shouldn't fail. %s", err)
	}
}

func TestUnitExecSycNotRunning(t *testing.T) {
	service := NewFakeCRIService(false)
	containerId, err := createContaier(FAKECONTAINERID, service)
	if err != nil {
		t.Fatal(err)
	}
	command := []string{"echo", "hola"}
	req := NewExecSyncRequest(containerId, command)
	_, err = service.ExecSync(nil, &req)
	if err == nil {
		t.Errorf("Exec should fail. %s", err)
	}
}

func TestUnitExecSyc(t *testing.T) {
	service := NewFakeCRIService(false)
	containerId, err := createContaier(FAKECONTAINERID, service)
	if err != nil {
		t.Fatal(err)
	}
	reqStart := NewContainerStartRequest(containerId)
	_, err = service.StartContainer(nil, &reqStart)
	if err != nil {
		t.Fatal(err)
	}
	command := []string{"echo", "hola"}
	req := NewExecSyncRequest(containerId, command)
	_, err = service.ExecSync(nil, &req)
	if err != nil {
		t.Errorf("Exec fails. %s", err)
	}
}
