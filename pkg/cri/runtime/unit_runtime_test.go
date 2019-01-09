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

	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

//Test retrieve cri version
func TestUnitVersion(t *testing.T) {
	service := NewFakeCRIService(false)
	req := runtimeapi.VersionRequest{""}
	out, err := service.Version(nil, &req)
	if err != nil {
		t.Errorf("Get version fail")
	}
	if out.Version != FAKEVERSION {
		t.Errorf("Get version wrong")
	}
}

//Test cri status
func TestUnitStatus(t *testing.T) {
	service := NewFakeCRIService(false)
	req := runtimeapi.StatusRequest{}
	out, err := service.Status(nil, &req)
	if err != nil {
		t.Errorf("Get status fails")
	}
	if out.Status.Conditions[0].Type != runtimeapi.RuntimeReady {
		t.Errorf("Get runtime status wrong: %s", out.Status.Conditions[0].Type)
	}
	if out.Status.Conditions[1].Type != runtimeapi.NetworkReady {
		t.Errorf("Get network status wrong: %s", out.Status.Conditions[1].Type)
	}
}
