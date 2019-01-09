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

package store

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/docker/distribution/uuid"
	runtimeApi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

type SandboxStoreInterface interface {
	Add(sm *SandboxMetadata)
	Update(sm *SandboxMetadata)
	Remove(ID string)
	RemoveContainer(ID, containerId string)
	Get(ID string) (*SandboxMetadata, error)
	List() map[string]*SandboxMetadata
	ListK8s(filter *runtimeApi.PodSandboxFilter, locaCRI string) []*runtimeApi.PodSandbox
	CreateSandboxMetadata(state runtimeApi.PodSandboxState, config runtimeApi.PodSandboxConfig, runtimeHandler string) *SandboxMetadata
}

type SandboxStorage struct {
	lock        sync.RWMutex
	persist     *SandboxPersist
	SandboxPool map[string]*SandboxMetadata
}

func NewSandboxStorage(resourceCachePath string, enablePersistence bool) (SandboxStoreInterface, error) {
	ss := new(SandboxStorage)
	if enablePersistence {
		p, err := NewSandboxPersist(resourceCachePath) //todo manage error
		if err != nil {
			return nil, fmt.Errorf("Error when opening Sandbox %s", err.Error())
		}
		ss.persist = p
		ss.SandboxPool = p.LoadAll()
	} else {
		ss.SandboxPool = make(map[string]*SandboxMetadata)
	}
	return ss, nil
}

func (ss *SandboxStorage) List() map[string]*SandboxMetadata {
	ss.lock.Lock()
	defer ss.lock.Unlock()
	return ss.SandboxPool
}

func (ss *SandboxStorage) Add(sm *SandboxMetadata) {
	ss.lock.Lock()
	defer ss.lock.Unlock()
	if _, exists := ss.SandboxPool[sm.ID]; exists {
		fmt.Errorf("Sandbox %s already exits", sm.ID)
	}
	ss.SandboxPool[sm.ID] = sm
	if ss.persist != nil {
		ss.persist.Put(sm.ID, sm)
	}
}

func (ss *SandboxStorage) Update(sm *SandboxMetadata) {
	ss.lock.Lock()
	defer ss.lock.Unlock()
	ss.SandboxPool[sm.ID] = sm
	if ss.persist != nil {
		ss.persist.Put(sm.ID, sm)
	}
}

func (ss *SandboxStorage) Remove(ID string) {
	ss.lock.Lock()
	defer ss.lock.Unlock()
	delete(ss.SandboxPool, ID)
	if ss.persist != nil {
		ss.persist.Delete(ID)
	}
}

func (ss *SandboxStorage) RemoveContainer(ID, containerId string) {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	sm := ss.SandboxPool[ID]
	if sm != nil {
		ss.SandboxPool[sm.ID] = sm
		for i, v := range sm.Containers {
			if v == containerId {
				sm.Containers = append(sm.Containers[:i], sm.Containers[i+1:]...)
			}
		}
		ss.SandboxPool[sm.ID] = sm
		if ss.persist != nil {
			ss.persist.Put(sm.ID, sm)
		}
	}
}

func (ss *SandboxStorage) Get(ID string) (*SandboxMetadata, error) {
	ss.lock.Lock()
	defer ss.lock.Unlock()
	value, ok := ss.SandboxPool[ID]
	if !ok {
		return nil, fmt.Errorf("Pod not found")
	}
	return value, nil
}

func (ss *SandboxStorage) ListK8s(filter *runtimeApi.PodSandboxFilter, localCRI string) []*runtimeApi.PodSandbox {
	ss.lock.Lock()
	defer ss.lock.Unlock()
	var result []*runtimeApi.PodSandbox
	filterPodID := ""
	filterPodUID := ""
	filterState := runtimeApi.PodSandboxState(100)
	if filter != nil {
		if filter.State != nil {
			filterState = filter.GetState().State
		}
		filterPodID = filter.Id
		if filter.LabelSelector != nil {
			for k, v := range filter.LabelSelector {
				if k == "io.kubernetes.pod.uid" {
					filterPodUID = v
				}
			}
		}
	}
	for _, sandbox := range ss.SandboxPool {
		if filterPodID != "" && sandbox.ID != filterPodID {
			continue
		}
		if filterPodUID != "" && sandbox.ID != filterPodUID {
			continue
		}
		if filterState < 100 && filterState != sandbox.State {
			continue
		}
		if sandbox.RuntimeHandler != localCRI {
			continue
		}
		result = append(result, ParseToK8sSandbox(sandbox))
	}
	return result
}

func getPodID(config runtimeApi.PodSandboxConfig) string {
	var ID string
	if len(config.GetMetadata().Uid) > 0 {
		ID = config.GetMetadata().Uid
	} else {
		ID = strings.Replace(uuid.Generate().String(), "-", "", -1)
	}
	return ID
}

func (ss *SandboxStorage) CheckIfExist(config runtimeApi.PodSandboxConfig) (*SandboxMetadata, error) {
	ID := getPodID(config)
	return ss.Get(ID)
}

func (ss *SandboxStorage) CreateSandboxMetadata(state runtimeApi.PodSandboxState, config runtimeApi.PodSandboxConfig,
	runtimeHandler string) *SandboxMetadata {
	ID := getPodID(config)

	createdAt := int64(time.Now().UnixNano())
	cgroup := ""
	if config.Linux != nil {
		cgroup = config.Linux.CgroupParent
	}
	sm := SandboxMetadata{ID: ID, State: state, Config: config, CreatedAt: createdAt, CgroupsParent: cgroup,
		RuntimeHandler: runtimeHandler}
	ss.Add(&sm)
	return &sm
}

// converts sandbox metadata into CRI pod sandbox.
func ParseToK8sSandbox(sandbox *SandboxMetadata) *runtimeApi.PodSandbox {
	ID := sandbox.ID
	config := sandbox.Config
	return &runtimeApi.PodSandbox{
		Id:          ID,
		Metadata:    config.GetMetadata(),
		State:       sandbox.State,
		CreatedAt:   sandbox.CreatedAt,
		Labels:      config.GetLabels(),
		Annotations: config.GetAnnotations(),
	}
}

// converts sandbox metadata into CRI pod sandbox status.
func ParseToK8sSandboxStatus(sandbox *SandboxMetadata, ip string) *runtimeApi.PodSandboxStatus {
	config := sandbox.Config
	nsOpts := config.GetLinux().GetSecurityContext().GetNamespaceOptions()
	return &runtimeApi.PodSandboxStatus{
		Id:        sandbox.ID,
		Metadata:  config.GetMetadata(),
		State:     sandbox.State,
		CreatedAt: sandbox.CreatedAt,
		Network:   &runtimeApi.PodSandboxNetworkStatus{Ip: ip},
		Linux: &runtimeApi.LinuxPodSandboxStatus{
			Namespaces: &runtimeApi.Namespace{
				Options: nsOpts,
			},
		},
		Labels:      config.GetLabels(),
		Annotations: config.GetAnnotations(),
	}

}
