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
	"fmt"
	"io"
	"os"

	"multi-cri/pkg/cri/network"
	"multi-cri/pkg/cri/store"

	"multi-cri/pkg/cri/runtime/remote"

	"github.com/cri-o/ocicni/pkg/ocicni"
	"golang.org/x/net/context"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
	"k8s.io/kubernetes/pkg/kubelet/server/streaming"
)

const (
	FAKECONTAINERID_RUNNING = "containerRunning"
	FAKECONTAINERID         = "containerCreated"
	FAKEVERSION             = "0.1"
	FAKESANDBOXID           = "pod1"
	NOEXISTSANDBOXID        = "wrongpodID"
	FAKEIMAGE_DOCKER        = store.CRIDockerRepository + "server:v1"
	FAKEIMAGE_HUB           = store.CRISingularityHubRepository + "server:v1"
	FAKEIMAGE_LOCAL         = store.CRISLocalRepository + "server:v1"
	FAKEIMAGE_DEF_FILE      = store.CRISLocalDefinitionFile + "server.def:v1"
)

//FAKE NETWORK
type FakeCNIPlugin struct {
}

func (*FakeCNIPlugin) Name() string {
	return "name"
}
func (*FakeCNIPlugin) SetUpPod(network ocicni.PodNetwork) error {
	return nil
}
func (*FakeCNIPlugin) TearDownPod(network ocicni.PodNetwork) error {
	return nil
}
func (*FakeCNIPlugin) GetPodNetworkStatus(network ocicni.PodNetwork) (string, error) {
	return "OK", nil
}
func (*FakeCNIPlugin) Status() error {
	return nil
}

//Network Namespace manager
type FakeNetworkNamespace struct{}

func (f *FakeNetworkNamespace) CreateNetNS(path string) error {
	return nil
}
func (f *FakeNetworkNamespace) Remove() error {
	return nil
}
func (f *FakeNetworkNamespace) GetPath() string {
	return ""
}

func NewFackeCNIPlugin() ocicni.CNIPlugin {
	return &FakeCNIPlugin{}
}

//
type FakeNetworkManager struct{}

func (f *FakeNetworkManager) OpenNetNamespace(path string) (network.NetworkNamespaceInterface, error) {
	net := new(FakeNetworkNamespace)
	return net, nil
}

func newFakeStreamServer() streaming.Server {
	config := streaming.DefaultConfig
	out, _ := streaming.NewServer(config, nil)
	return out
}

//FAKE SINGULARITY SERVICE
func NewFakeCRIService(adapterFails bool) CRIMulticriService {
	iStorage, err := store.NewImageStorage("", false)
	if err != nil {
		panic(err)
	}
	containerStorage, err := store.NewContainerStorage("", false)
	if err != nil {
		panic(err)
	}
	sandboxStorage, err := store.NewSandboxStorage("", false)
	if err != nil {
		panic(err)
	}
	f := multicriService{
		netPlugin:        NewFackeCNIPlugin(),
		sandboxStore:     sandboxStorage,
		containerStore:   containerStorage,
		imageStore:       iStorage,
		adapter:          &FakeAdapter{fails: adapterFails},
		networkNamespace: &FakeNetworkManager{},
		os:               &FakeOS{},
		streamServer:     newFakeStreamServer(),
		remoteCRI:        &remote.RemoteCRIConfiguration{},
	}

	multicriRuntime := NewMulticriRuntime(&f)
	return multicriRuntime
}

//////////////////
//Fake requests///
//////////////////

// Request sandbox
func NewCreateSandboxRequest(uid, runtimeHandler string) runtimeapi.RunPodSandboxRequest {
	return runtimeapi.RunPodSandboxRequest{
		Config: &runtimeapi.PodSandboxConfig{
			Metadata: &runtimeapi.PodSandboxMetadata{Uid: uid},
		},
		RuntimeHandler: runtimeHandler,
	}
}

func NewStopSandboxRequest(podId string) runtimeapi.StopPodSandboxRequest {
	return runtimeapi.StopPodSandboxRequest{podId}
}

func NewRemoveSandboxRequest(podId string) runtimeapi.RemovePodSandboxRequest {
	return runtimeapi.RemovePodSandboxRequest{podId}
}

func NewStatusSandboxRequest(podId string) runtimeapi.PodSandboxStatusRequest {
	return runtimeapi.PodSandboxStatusRequest{podId, false}
}

func NewListSandboxRequest(filter runtimeapi.PodSandboxFilter) runtimeapi.ListPodSandboxRequest {
	return runtimeapi.ListPodSandboxRequest{&filter}
}
func NewPortForwardRequest(podId string, port []int32) runtimeapi.PortForwardRequest {
	return runtimeapi.PortForwardRequest{podId, port}
}

//Request containers//
func NewCreateContainerRequest(podID, name, image string) runtimeapi.CreateContainerRequest {
	return runtimeapi.CreateContainerRequest{PodSandboxId: podID,
		Config: &runtimeapi.ContainerConfig{Metadata: &runtimeapi.ContainerMetadata{Name: name},
			Image: &runtimeapi.ImageSpec{Image: image}},
		SandboxConfig: &runtimeapi.PodSandboxConfig{LogDirectory: "/tmp",
			Labels: map[string]string{}},
	}
}

func NewContainerStatusRequest(containerID string) runtimeapi.ContainerStatusRequest {
	return runtimeapi.ContainerStatusRequest{ContainerId: containerID}
}

func NewContainerRemoveRequest(containerID string) runtimeapi.RemoveContainerRequest {
	return runtimeapi.RemoveContainerRequest{ContainerId: containerID}
}

func NewContainerStartRequest(containerID string) runtimeapi.StartContainerRequest {
	return runtimeapi.StartContainerRequest{ContainerId: containerID}
}

func NewContainerStopRequest(containerID string) runtimeapi.StopContainerRequest {
	return runtimeapi.StopContainerRequest{ContainerId: containerID}
}

func NewListContainersRequest(filter runtimeapi.ContainerFilter) runtimeapi.ListContainersRequest {
	req := runtimeapi.ListContainersRequest{&filter}
	return req
}

func NewContainerStatsRequest(containerID string) runtimeapi.ContainerStatsRequest {
	req := runtimeapi.ContainerStatsRequest{ContainerId: containerID}
	return req
}

func NewListContainerStatsRequest(filter runtimeapi.ContainerStatsFilter) runtimeapi.ListContainerStatsRequest {
	req := runtimeapi.ListContainerStatsRequest{Filter: &filter}
	return req
}

///Exec Requests
func NewAttachRequest(containerID string) runtimeapi.AttachRequest {
	return runtimeapi.AttachRequest{ContainerId: containerID}
}

func NewExecRequest(containerID string, cmd []string) runtimeapi.ExecRequest {
	return runtimeapi.ExecRequest{ContainerId: containerID, Cmd: cmd, Stdin: true}
}

func NewExecSyncRequest(containerID string, cmd []string) runtimeapi.ExecSyncRequest {
	return runtimeapi.ExecSyncRequest{ContainerId: containerID, Cmd: cmd}
}

//Image
func NewListImagesRequest(filter runtimeapi.ImageFilter) runtimeapi.ListImagesRequest {
	return runtimeapi.ListImagesRequest{Filter: &filter}
}

func NewImageStatusRequest(image string) runtimeapi.ImageStatusRequest {
	spec := runtimeapi.ImageSpec{Image: image}
	return runtimeapi.ImageStatusRequest{&spec, false}
}

func NewImageFsInfoRequest() runtimeapi.ImageFsInfoRequest {
	return runtimeapi.ImageFsInfoRequest{}
}

func NewPullImageRequest(image string, podId string, labels map[string]string) runtimeapi.PullImageRequest {
	spec := runtimeapi.ImageSpec{Image: image}
	conf := runtimeapi.PodSandboxConfig{
		Metadata: &runtimeapi.PodSandboxMetadata{Uid: podId},
		Labels:   labels,
	}
	return runtimeapi.PullImageRequest{Image: &spec, SandboxConfig: &conf}
}

func downloadFileFake(destiny string, source string) (err error) {
	return nil
}

func NewRemoveImageRequest(image string) runtimeapi.RemoveImageRequest {
	spec := runtimeapi.ImageSpec{Image: image}
	return runtimeapi.RemoveImageRequest{&spec}
}

type FakeOS struct{}

func (f *FakeOS) MkdirAll(path string, perm os.FileMode) error { return nil }
func (f *FakeOS) RemoveAll(path string) error                  { return nil }
func (f *FakeOS) OpenFifo(_ context.Context, fn string, flag int, perm os.FileMode) (io.ReadWriteCloser, error) {
	return nil, nil
}
func (f *FakeOS) Stat(name string) (os.FileInfo, error)                          { return nil, nil }
func (f *FakeOS) CopyFile(src, dest string, perm os.FileMode) error              { return nil }
func (f *FakeOS) WriteFile(filename string, data []byte, perm os.FileMode) error { return nil }
func (f *FakeOS) Mount(source string, target string, fstype string, flags uintptr, data string) error {
	return nil
}
func (f *FakeOS) Unmount(target string, flags int) error { return nil }
func (f *FakeOS) IsNotExist(err error) bool              { return true }

//////////////////
//Fake Adapter ///
//////////////////

type FakeAdapter struct {
	fails bool
}

func (f *FakeAdapter) CreateContainer(cm *store.ContainerMetadata) error { return nil }
func (f *FakeAdapter) StartContainer(cm *store.ContainerMetadata) error  { return nil }
func (f *FakeAdapter) StopContainer(cm *store.ContainerMetadata) error {
	if f.fails {
		return fmt.Errorf("Adapter fails")
	}
	return nil
}
func (f *FakeAdapter) ContainerStatus(cm *store.ContainerMetadata) error          { return nil }
func (r *FakeAdapter) ReopenContainerLog(cm *store.ContainerMetadata) error       { return nil }
func (r *FakeAdapter) UpdateContainerResources(cm *store.ContainerMetadata) error { return nil }
func (f *FakeAdapter) ExecSync(cm *store.ContainerMetadata, command []string) (*runtimeapi.ExecSyncResponse, error) {
	return &runtimeapi.ExecSyncResponse{ExitCode: 1}, nil
}
func (f *FakeAdapter) Exec(cm *store.ContainerMetadata, req *runtimeapi.ExecRequest) (*runtimeapi.ExecResponse, error) {
	return &runtimeapi.ExecResponse{}, nil
}

func (f *FakeAdapter) Attach(cm *store.ContainerMetadata, req *runtimeapi.AttachRequest) (*runtimeapi.AttachResponse, error) {
	return nil, nil
}

func (f *FakeAdapter) PullImage(image *store.ImageMetadata) error {
	image.LocalPath = "/tmp"
	return nil
}
func (f *FakeAdapter) ListImages(images []*runtimeapi.Image) error  { return nil }
func (f *FakeAdapter) ImageStatus(image *store.ImageMetadata) error { return nil }
func (f *FakeAdapter) ImageFsInfo() (*runtimeapi.ImageFsInfoResponse, error) {
	return nil, fmt.Errorf("ImageFsInfo still not implemented")
}
func (f *FakeAdapter) RemoveImage(image *store.ImageMetadata) error { return nil }

func (f *FakeAdapter) NewStreamRuntime(c store.ContainerStoreInterface) streaming.Runtime { return nil }

func (r *FakeAdapter) Version() (*runtimeapi.VersionResponse, error) {
	return &runtimeapi.VersionResponse{Version: FAKEVERSION}, nil
}
func (r *FakeAdapter) RunPodSandbox(sandbox *store.SandboxMetadata) error    { return nil }
func (r *FakeAdapter) StopPodSandbox(sandbox *store.SandboxMetadata) error   { return nil }
func (r *FakeAdapter) RemovePodSandbox(sandbox *store.SandboxMetadata) error { return nil }
func (r *FakeAdapter) PodSandboxStatus(sandbox *store.SandboxMetadata) error { return nil }
