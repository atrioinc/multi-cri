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
	"cri-babelfish/pkg/cri/common/file"
	"fmt"
	"strings"
	"sync"

	"github.com/docker/distribution/uuid"
	"k8s.io/klog"
	runtimeApi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

const (
	CRISingularityHubRepository             = "singularity-repository."
	CRIDockerRepository                     = "docker-repository."
	CRISLocalRepository                     = "local-image."
	CRISLocalDefinitionFile                 = "local-def-file."
	SingularityRepositoryImageRepo RepoType = 0
	DockerRepositoryImageRepo      RepoType = 1
	LocalImageRepo                 RepoType = 2
	LocalDefinitionFile            RepoType = 3
	UnknownImageRepo               RepoType = 4
)

type ImageStoreInterface interface {
	List() map[string]*ImageMetadata
	Add(sm *ImageMetadata) error
	Update(sm *ImageMetadata)
	Remove(ID string)
	Get(ID string) (*ImageMetadata, error)
	GetByID(ID string) (*ImageMetadata, error)
	GetByPath(path string) (*ImageMetadata, error)
	CreateImageMetadata(imageName *runtimeApi.PullImageRequest) (*ImageMetadata, error)
	ListK8s(filter *runtimeApi.ImageFilter) []*runtimeApi.Image
}

type ImageStorage struct {
	lock sync.RWMutex
	// base local path
	basePath string
	//persist
	persist *ImagePersist
	// image pool
	ImagePool map[string]*ImageMetadata
}

func NewImageStorage(resourceCachePath string, enablePersistence bool) (ImageStoreInterface, error) {
	im := new(ImageStorage)
	if enablePersistence {
		im.basePath = resourceCachePath
		p, err := NewImagePersist(resourceCachePath)
		if err != nil {
			return nil, fmt.Errorf("Error when opening Image Cache: %s", err)
		}
		im.persist = p
		im.ImagePool = p.LoadAll()
	} else {
		im.ImagePool = make(map[string]*ImageMetadata)
	}
	return im, nil
}

func (im *ImageStorage) List() map[string]*ImageMetadata {
	im.lock.Lock()
	defer im.lock.Unlock()
	return im.ImagePool
}

func (im *ImageStorage) Add(sm *ImageMetadata) error {
	im.lock.Lock()
	defer im.lock.Unlock()
	if _, exists := im.ImagePool[sm.ID]; exists {
		//Fixme: See what to do here
		return fmt.Errorf("Image %s already exits", sm.ID)
	}
	//todo check if exists
	im.ImagePool[sm.ID] = sm
	if im.persist != nil {
		im.persist.Put(sm.ID, sm)
	}
	return nil
}

func (im *ImageStorage) Update(sm *ImageMetadata) {
	im.lock.Lock()
	defer im.lock.Unlock()
	im.ImagePool[sm.ID] = sm
	if im.persist != nil {
		im.persist.Put(sm.ID, sm)
	}
}

func (im *ImageStorage) Remove(ID string) {
	im.lock.Lock()
	defer im.lock.Unlock()
	im.remove(ID)
}

func (im *ImageStorage) remove(ID string) {
	delete(im.ImagePool, ID)
	if im.persist != nil {
		im.persist.Delete(ID)
	}
}
func (im *ImageStorage) Get(ID string) (*ImageMetadata, error) {
	im.lock.Lock()
	defer im.lock.Unlock()
	for imageId, imageData := range im.ImagePool {
		if strings.Contains(imageId, ID) { //using short id
			return imageData, nil
		}
	}
	return nil, fmt.Errorf("Image does not exists")
}

func (im *ImageStorage) GetByID(ID string) (*ImageMetadata, error) {
	im.lock.Lock()
	defer im.lock.Unlock()
	var imageFound *ImageMetadata
	for imageId, image := range im.ImagePool {
		if strings.Contains(image.ImageName, ID) || strings.Contains(imageId, ID) || tagInImage(image, ID) || digestInImage(image, ID) {
			imageFound = image
			break
		}
	}
	if imageFound == nil {
		return nil, fmt.Errorf("Image does not exists")
	} else {
		return imageFound, nil
	}
}
func tagInImage(image *ImageMetadata, tag string) bool {
	for _, t := range image.RepoTags {
		if t == tag {
			return true
		}
	}
	return false
}

func digestInImage(image *ImageMetadata, digest string) bool {
	for _, d := range image.RepoDigest {
		if d == digest {
			return true
		}
	}
	return false
}

func (im *ImageStorage) GetByPath(path string) (*ImageMetadata, error) {
	im.lock.Lock()
	defer im.lock.Unlock()
	var imageFound *ImageMetadata
	for _, image := range im.ImagePool {
		if image.LocalPath == path || image.RemotePath == path {
			imageFound = image
			break
		}
	}
	if imageFound == nil {
		return nil, fmt.Errorf("Image does not exists")
	} else {
		return imageFound, nil
	}
}

func (im *ImageStorage) CreateImageMetadata(req *runtimeApi.PullImageRequest) (*ImageMetadata, error) {
	uuidString := strings.Replace(uuid.Generate().String(), "-", "", -1)
	var auth ImageAuth
	if req.GetAuth() != nil {
		auth.Username = req.GetAuth().Username
		auth.Password = req.GetAuth().Password
		auth.IdentityToken = req.GetAuth().IdentityToken
		auth.RegistryToken = req.GetAuth().RegistryToken
		auth.ServerAddress = req.GetAuth().ServerAddress
	}
	image := &ImageMetadata{
		RemotePath: req.Image.Image,
		ID:         uuidString,
		Auth:       auth,
		LocalPath:  im.basePath + "/.images",
	}
	im.Add(image)
	return image, nil
}

func (im *ImageStorage) ListK8s(filter *runtimeApi.ImageFilter) []*runtimeApi.Image {
	im.lock.Lock()
	defer im.lock.Unlock()
	var imageArray []*runtimeApi.Image
	for _, image := range im.ImagePool {
		if filter.GetImage() != nil && len(filter.GetImage().Image) > 0 {
			if image.RemotePath == filter.GetImage().Image || digestInImage(image, filter.GetImage().Image) {
				//Ensure image file exits
				if !file.CheckFileExist(image.LocalPath) {
					klog.V(4).Infof("Image %s removed from list because its file does not exist", image.ID)
					im.remove(image.ID)
				} else {
					imageArray = append(imageArray, ParseToK8sImage(image))
				}
			}
		} else {
			//Ensure image file exits
			if !file.CheckFileExist(image.LocalPath) {
				klog.V(4).Infof("Image %s removed from list because its file does not exist", image.ID)
				im.remove(image.ID)
			} else {
				imageArray = append(imageArray, ParseToK8sImage(image))
			}
		}
	}
	return imageArray
}

func ParseToK8sImage(img *ImageMetadata) *runtimeApi.Image {
	out := runtimeApi.Image{
		Id: img.RemotePath, Size_: img.Size,
		RepoDigests: img.RepoDigest, RepoTags: img.RepoTags,
	}
	return &out
}

//Parse singularity hub image
func ParseImage(image *ImageMetadata) error {
	var typeImage, urlImage string
	var repoType RepoType
	if strings.HasPrefix(image.RemotePath, CRISingularityHubRepository) {
		typeImage = CRISingularityHubRepository
		urlImage = "shub://"
		repoType = SingularityRepositoryImageRepo
	} else if strings.HasPrefix(image.RemotePath, CRIDockerRepository) {
		typeImage = CRIDockerRepository
		urlImage = "docker://"
		repoType = DockerRepositoryImageRepo
	} else if strings.HasPrefix(image.RemotePath, CRISLocalRepository) {
		typeImage = CRISLocalRepository
		urlImage = ""
		repoType = LocalImageRepo

	} else if strings.HasPrefix(image.RemotePath, CRISLocalDefinitionFile) {
		typeImage = CRISLocalDefinitionFile
		urlImage = ""
		repoType = LocalDefinitionFile
	} else {
		return fmt.Errorf("Image repository is not supported " + image.RemotePath)

	}
	imageSplited := strings.Split(image.RemotePath, typeImage)
	realImagePath := imageSplited[len(imageSplited)-1]
	image.RepoType = repoType
	image.RepoDigest = []string{image.RemotePath}
	splitImage := strings.Split(realImagePath, "/")
	nameWithVersion := splitImage[len(splitImage)-1]
	image.ImageName = strings.Split(nameWithVersion, ":")[0]

	image.RemotePath = urlImage + realImagePath
	repoTag := []string{image.RemotePath}
	image.RepoTags = repoTag
	return nil
}

func (im *ImageStorage) GetBasePath() string {
	return im.basePath
}
