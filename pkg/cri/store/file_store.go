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
	"reflect"

	"cri-babelfish/pkg/cri/network"

	"github.com/jorgesece/skv"

	"k8s.io/klog"
)

const (
	CONTAINERSTORE = "cache-cri-containers.dat"
	SANDBOXSTORE   = "cache-cri-sandboxes.dat"
	IMAGESTORE     = "cache-cri-images.dat"
)

type ContainerPersist struct {
	StoreDriver *skv.KVStore
	Path        string
}

func NewContainerPersist(resourceCachePath string) (*ContainerPersist, error) {
	path := resourceCachePath + "/" + CONTAINERSTORE
	storeDriver, err := skv.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Error openning file %s.", path)
	}
	cp := &ContainerPersist{StoreDriver: storeDriver, Path: path}
	return cp, nil
}

//LoadAll loads all the containers stored in file
func (fp *ContainerPersist) LoadAll() (out map[string]*ContainerMetadata) {
	klog.V(4).Infof("Loading containers from with file storage %s", fp.Path)
	val := ContainerMetadata{}
	out = make(map[string]*ContainerMetadata)
	result, err := fp.StoreDriver.List(reflect.TypeOf(val))
	for k, v := range result {
		out[k] = v.(*ContainerMetadata)
	}
	if err != nil {
		klog.Errorf("Error when loading containers from file %s", fp.Path)
	}
	return out
}

// get: fetches from boltdb and does gob decode
func (fp *ContainerPersist) Get(containerId string, containerObject *ContainerMetadata) error {
	err := fp.StoreDriver.Get(containerId, containerObject)
	if err != nil {
		klog.Errorf("Error when getting container %s in file", containerId)
	}
	return err
}

// put: encodes value with gob and updates the boltdb
func (fp *ContainerPersist) Put(containerId string, containerObject *ContainerMetadata) error {
	err := fp.StoreDriver.Put(containerId, containerObject)
	if err != nil {
		klog.Errorf("Error when storing container %s in file", containerId)
	}
	return err
}

// delete: seeks in boltdb and deletes the record
func (fp *ContainerPersist) Delete(containerId string) error {
	err := fp.StoreDriver.Delete(containerId)
	if err != nil {
		klog.Errorf("Error when deleting container %s in file", containerId)
	}
	return err
}

// close the store
func (fp *ContainerPersist) Close() error {
	err := fp.StoreDriver.Close()
	if err != nil {
		klog.Errorf("Error when closing container persist from file %s", fp.Path)
	}
	return err
}

//SANDBOX
type SandboxPersist struct {
	StoreDriver *skv.KVStore
	Path        string
	NetworkNS   network.NetworkManagerInterface
}

func NewSandboxPersist(resourceCachePath string) (*SandboxPersist, error) {
	path := resourceCachePath + "/" + SANDBOXSTORE
	storeDriver, err := skv.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Error openning file %s.", path)
	}
	cp := &SandboxPersist{StoreDriver: storeDriver, Path: path, NetworkNS: network.NewNetworkCNIManager()}
	return cp, nil
}

func (fp *SandboxPersist) LoadAll() (out map[string]*SandboxMetadata) {
	klog.V(4).Infof("Loading sandboxes from with file storage %s", fp.Path)
	val := SandboxMetadata{}
	out = make(map[string]*SandboxMetadata)
	result, err := fp.StoreDriver.List(reflect.TypeOf(val))
	for k, v := range result {
		value := v.(*SandboxMetadata)
		if value.NetNSPath != "" && value.State == 0 {
			ns, err := fp.NetworkNS.OpenNetNamespace(value.NetNSPath)
			if err != nil {
				klog.Errorf("Error when loading namespace of sandbox %s", value.ID)
			} else {
				value.NetNSPath = ns.GetPath()
				fp.Put(k, value)
				out[k] = value
			}
		} else {
			fp.Delete(k)
		}
	}
	if err != nil {
		klog.Errorf("Error when loading sandboxes from file %s", fp.Path)
	}
	return out
}

// get: fetches from boltdb and does gob decode
func (fp *SandboxPersist) Get(SandboxId string, SandboxObject *SandboxMetadata) error {
	err := fp.StoreDriver.Get(SandboxId, SandboxObject)
	if err != nil {
		klog.Errorf("Error when getting Sandbox %s in file", SandboxId)
	}
	return err
}

// put: encodes value with gob and updates the boltdb
func (fp *SandboxPersist) Put(SandboxId string, SandboxObject *SandboxMetadata) error {
	err := fp.StoreDriver.Put(SandboxId, SandboxObject)
	if err != nil {
		klog.Errorf("Error when storing Sandbox %s in file", SandboxId)
	}
	return err
}

// delete: seeks in boltdb and deletes the record
func (fp *SandboxPersist) Delete(SandboxId string) error {
	err := fp.StoreDriver.Delete(SandboxId)
	if err != nil {
		klog.Errorf("Error when deleting Sandbox %s in file", SandboxId)
	}
	return err
}

// close the store
func (fp *SandboxPersist) Close() error {
	err := fp.StoreDriver.Close()
	if err != nil {
		klog.Errorf("Error when closing Sandbox persist from file %s", fp.Path)
	}
	return err
}

//IMAGE PERSIST
type ImagePersist struct {
	StoreDriver *skv.KVStore
	Path        string
}

func NewImagePersist(resourceCachePath string) (*ImagePersist, error) {
	path := resourceCachePath + "/" + IMAGESTORE
	storeDriver, err := skv.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Error openning file %s.", path)
	}
	cp := &ImagePersist{StoreDriver: storeDriver, Path: path}
	return cp, nil
}

//LoadAll loads all the images stored in file
func (fp *ImagePersist) LoadAll() (out map[string]*ImageMetadata) {
	klog.V(4).Infof("Loading images from with file storage %s", fp.Path)
	val := ImageMetadata{}
	out = make(map[string]*ImageMetadata)
	result, err := fp.StoreDriver.List(reflect.TypeOf(val))
	for k, v := range result {
		out[k] = v.(*ImageMetadata)
		//Todo pull if not exists in file?
	}
	if err != nil {
		klog.Errorf("Error when loading images from file %s", fp.Path)
	}
	return out
}

// get: fetches from boltdb and does gob decode
func (fp *ImagePersist) Get(ImageId string, ImageObject *ImageMetadata) error {
	err := fp.StoreDriver.Get(ImageId, ImageObject)
	if err != nil {
		klog.Errorf("Error when getting Image %s in file", ImageId)
	}
	return err
}

// put: encodes value with gob and updates the boltdb
func (fp *ImagePersist) Put(ImageId string, ImageObject *ImageMetadata) error {
	err := fp.StoreDriver.Put(ImageId, ImageObject)
	if err != nil {
		klog.Errorf("Error when storing Image %s in file", ImageId)
	}
	return err
}

// delete: seeks in boltdb and deletes the record
func (fp *ImagePersist) Delete(ImageId string) error {
	err := fp.StoreDriver.Delete(ImageId)
	if err != nil {
		klog.Errorf("Error when deleting Image %s in file", ImageId)
	}
	return err
}

// close the store
func (fp *ImagePersist) Close() error {
	err := fp.StoreDriver.Close()
	if err != nil {
		klog.Errorf("Error when closing Image persist from file %s", fp.Path)
	}
	return err
}
