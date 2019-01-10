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

package slurm

import (
	"multi-cri/pkg/cri/store"
	"fmt"

	runtimeApi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

func (s SlurmAdapter) ExecSync(cm *store.ContainerMetadata, command []string) (*runtimeApi.ExecSyncResponse, error) {
	return nil, fmt.Errorf("ExecSync not implemented for Slurm Adapter")
}
func (s SlurmAdapter) Exec(cm *store.ContainerMetadata, req *runtimeApi.ExecRequest) (*runtimeApi.ExecResponse, error) {
	return nil, fmt.Errorf("Exec not implemented for Slurm Adapter")
}

func (s SlurmAdapter) Attach(cm *store.ContainerMetadata, req *runtimeApi.AttachRequest) (*runtimeApi.AttachResponse, error) {
	return nil, fmt.Errorf("Attach not implemented for Slurm Adapter")
}
