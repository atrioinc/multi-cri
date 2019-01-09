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

	"github.com/docker/distribution/uuid"
	runtimeApi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

type ContainerStoreInterface interface {
	Add(sm *ContainerMetadata)
	Update(sm *ContainerMetadata)
	Remove(ID string)
	Get(ID string) (*ContainerMetadata, error)
	List(Id string, filterPodSandboxId string) []*ContainerMetadata
	ListK8s(Id string, filterPodSandboxId string, filterLabelSelector map[string]string,
		filterState *runtimeApi.ContainerStateValue, localCRI string) []*runtimeApi.Container
	CreateContainerMetadata(name string, podSandbox *SandboxMetadata, state runtimeApi.ContainerState,
		createdAt int64, image *ImageMetadata, command []string, args []string, isService bool,
		config runtimeApi.ContainerConfig, envVars map[string]string, port int, id *string,
	) *ContainerMetadata
}

type ContainerStorage struct {
	lock          sync.RWMutex
	persist       *ContainerPersist
	ContainerPool map[string]*ContainerMetadata
}

func NewContainerStorage(resourceCache string, enablePersistence bool) (ContainerStoreInterface, error) {
	cs := new(ContainerStorage)
	if enablePersistence {
		p, err := NewContainerPersist(resourceCache)
		if err != nil {
			return nil, fmt.Errorf("Error when opening %s", err.Error())
		}
		cs.persist = p
		cs.ContainerPool = p.LoadAll()
	} else {
		cs.ContainerPool = make(map[string]*ContainerMetadata)
	}
	return cs, nil
}

func (cs *ContainerStorage) List(Id string, filterPodSandboxId string) []*ContainerMetadata {
	cs.lock.Lock()
	defer cs.lock.Unlock()
	var containers []*ContainerMetadata
	for _, container := range cs.ContainerPool {
		if filterPodSandboxId != "" && container.PodSandbox.ID != filterPodSandboxId {
			continue
		}
		if Id != "" && Id != container.ID {
			continue
		}
		containers = append(containers, container)
	}
	return containers
}

func (cs *ContainerStorage) Add(cm *ContainerMetadata) {
	cs.lock.Lock()
	defer cs.lock.Unlock()
	if _, exists := cs.ContainerPool[cm.ID]; exists {
		fmt.Printf("Container %s already exits", cm.ID)
	}
	cs.ContainerPool[cm.ID] = cm
	if cs.persist != nil {
		cs.persist.Put(cm.ID, cm)
	}
}

func (cs *ContainerStorage) Update(cm *ContainerMetadata) {
	cs.lock.Lock()
	defer cs.lock.Unlock()
	cs.ContainerPool[cm.ID] = cm
	if cs.persist != nil {
		cs.persist.Put(cm.ID, cm)
	}
}

func (cs *ContainerStorage) Remove(ID string) {
	cs.lock.Lock()
	defer cs.lock.Unlock()
	delete(cs.ContainerPool, ID)
	if cs.persist != nil {
		cs.persist.Delete(ID)
	}
}

func (cs *ContainerStorage) Get(ID string) (*ContainerMetadata, error) {
	cs.lock.Lock()
	defer cs.lock.Unlock()
	value, ok := cs.ContainerPool[ID]
	if !ok {
		return nil, fmt.Errorf("Container not found")
	}
	return value, nil
}

func (cs *ContainerStorage) CreateContainerMetadata(name string, podSandbox *SandboxMetadata, state runtimeApi.ContainerState,
	createdAt int64, image *ImageMetadata, command []string, args []string, isService bool,
	config runtimeApi.ContainerConfig, envVars map[string]string, port int, id *string,
) *ContainerMetadata {
	var ID string
	if id != nil {
		ID = *id
	} else {
		ID = strings.Replace(uuid.Generate().String(), "-", "", -1)
	}
	containerLogPath := GenerateContainerLogPath(podSandbox.LogPath, ID)
	cm := ContainerMetadata{ID: ID, Name: name, State: state,
		Args: args, Config: config, CreatedAt: createdAt, LogFile: containerLogPath,
		IsService: isService, Command: command, Environment: envVars, Port: port, Extra: make(map[string]string)}
	if podSandbox != nil {
		cm.PodSandbox = *podSandbox
	}
	if image != nil {
		cm.Image = image
	}
	cs.Add(&cm)
	return &cm
}

func (cs *ContainerStorage) ListK8s(Id string, filterPodSandboxId string, filterLabelSelector map[string]string,
	filterState *runtimeApi.ContainerStateValue, localCRI string) []*runtimeApi.Container {
	cs.lock.Lock()
	defer cs.lock.Unlock()
	var containers []*runtimeApi.Container
	filterPodID := ""
	filterPodUID := ""
	filterStateLocal := runtimeApi.ContainerState(100)
	if filterState != nil {
		filterStateLocal = filterState.State
	}
	if filterPodSandboxId != "" {
		filterPodID = filterPodSandboxId
	}

	if filterLabelSelector != nil {
		for k, v := range filterLabelSelector {
			if k == "io.kubernetes.pod.uid" {
				filterPodUID = v
			}
		}
	}
	for _, container := range cs.ContainerPool {
		if filterPodID != "" && container.PodSandbox.ID != filterPodID {
			continue
		}
		if filterPodUID != "" && container.PodSandbox.ID != filterPodUID {
			continue
		}
		if filterStateLocal < 100 && filterStateLocal != container.State {
			continue
		}
		if Id != "" && Id != container.ID {
			continue
		}
		if container.PodSandbox.RuntimeHandler != localCRI {
			continue
		}
		containers = append(containers, ParseToK8sContainer(container))
	}
	return containers
}

func ParseToK8sContainer(cm *ContainerMetadata) *runtimeApi.Container {
	c := runtimeApi.Container{
		Id:           cm.ID,
		State:        cm.State,
		PodSandboxId: cm.PodSandbox.ID,
		Image:        &runtimeApi.ImageSpec{cm.Image.LocalPath},
		ImageRef:     cm.Image.LocalPath,
		Metadata:     cm.Config.Metadata,
		CreatedAt:    cm.CreatedAt,
		Labels:       cm.Config.Labels,
		Annotations:  cm.Config.Annotations,
	}
	return &c
}

func GetK8sContainerStatus(cm *ContainerMetadata) runtimeApi.ContainerStatus {
	status := runtimeApi.ContainerStatus{
		Id:          cm.ID,
		State:       cm.State,
		Image:       cm.Config.Image,
		ImageRef:    cm.Image.LocalPath,
		Metadata:    cm.Config.Metadata,
		StartedAt:   cm.StartedAt,
		FinishedAt:  cm.FinishedAt,
		CreatedAt:   cm.CreatedAt,
		LogPath:     cm.LogFile,
		Labels:      cm.Config.Labels,
		Annotations: cm.Config.Annotations,
		ExitCode:    int32(cm.ExitCode),
		Reason:      cm.Reason,
	}
	return status
}

func GenerateContainerLogPath(logPath, id string) string {
	return fmt.Sprintf("%s/%s.log", logPath, id)
}
