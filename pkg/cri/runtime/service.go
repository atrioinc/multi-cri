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
	"net"
	"os"
	"syscall"

	"cri-babelfish/pkg/cri/network"
	"cri-babelfish/pkg/cri/store"
	osinterface "cri-babelfish/pkg/os"

	"cri-babelfish/pkg/cri/adapters"
	slurmAdapter "cri-babelfish/pkg/cri/adapters/slurm"

	"cri-babelfish/pkg/cri/runtime/remote"

	"github.com/cri-o/ocicni/pkg/ocicni"
	"google.golang.org/grpc"
	"k8s.io/klog"
	runtimeapi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
	"k8s.io/kubernetes/pkg/kubelet/server/streaming"
)

const (
	// unixProtocol is the network protocol of unix socket.
	unixProtocol = "unix"
)

// CRISingularityService is the interface implement CRI remote service server.
type CRIBabelFishService interface {
	Run() error
	Stop()
	runtimeapi.RuntimeServiceServer
	runtimeapi.ImageServiceServer
}

// criBabelfishService implements CRIBabelfishService.
type criBabelfishService struct {
	// Exec path
	adapter adapters.AdapterInterface
	// serverAddress is the grpc server unix path.
	serverAddress string
	// server is the grpc server.
	server *grpc.Server
	// rootDir is the directory for managing cri-babelfish files.
	rootDir string
	// sandboxImage is the image to use for sandbox container.
	sandboxImage string
	// snapshotter is the snapshotter to use in babelfish.
	snapshotter string
	// sandboxStore stores all resources associated with sandboxes.
	sandboxStore store.SandboxStoreInterface
	// containerStore stores all resources associated with containers.
	containerStore store.ContainerStoreInterface
	// imageStore stores all resources associated with images.
	imageStore store.ImageStoreInterface
	// netPlugin is used to setup and teardown network when run/stop pod sandbox.
	netPlugin ocicni.CNIPlugin
	//NetworkNamespace Driver
	networkNamespace network.NetworkManagerInterface
	// streamServer is the streaming server serves container streaming request.
	streamServer streaming.Server
	// cgroupPath in which the cri is placed in
	cgroupPath string
	// os is an interface for all required os operations.
	os        osinterface.OS
	remoteCRI *remote.RemoteCRIConfiguration
}

func loadAdapter(adapterName string) (adapters.AdapterInterface, error) {
	if adapterName == "fake" {
		return &FakeAdapter{fails: false}, nil
	} else if adapterName == "slurm" {
		return slurmAdapter.NewSlurmAdapter()
	} else {
		return nil, fmt.Errorf("Adapter not found")
	}
}

func NewBabelFishService(
	adapterName,
	socketPath,
	networkPluginBinDir,
	networkPluginConfDir,
	streamAddress,
	streamPort,
	cgroupPath,
	sandboxImage,
	resourceCachePath string,
	enablePodPersistence bool,
	enableNetworkPersistence bool,
	remoteCRIEndpoints string,
) (CRIBabelFishService, error) {
	criAdapter, err := loadAdapter(adapterName)
	if err != nil {
		return nil, err
	}
	if cgroupPath != "" {
		_, err := loadCgroup(cgroupPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load cgroup for cgroup path %v: %v", cgroupPath, err)
		}
	}
	err = os.MkdirAll(resourceCachePath, 0755)
	if err != nil {
		klog.Errorf("Failed to create resource cache directory %s", resourceCachePath)
	}
	sandboxStore, err := store.NewSandboxStorage(resourceCachePath, enablePodPersistence)
	if err != nil {
		return nil, err
	}
	containerStore, err := store.NewContainerStorage(resourceCachePath, enablePodPersistence)
	if err != nil {
		return nil, err
	}
	imageStore, err := store.NewImageStorage(resourceCachePath, true)
	if err != nil {
		return nil, err
	}

	remoteCRI, err := remote.LoadRemoteRuntimeConfiguration(remoteCRIEndpoints)
	if err != nil {
		return nil, err
	}
	c := &criBabelfishService{
		serverAddress:    socketPath,
		sandboxImage:     sandboxImage,
		networkNamespace: network.NewNetworkCNIManager(),
		sandboxStore:     sandboxStore,
		containerStore:   containerStore,
		imageStore:       imageStore,
		adapter:          criAdapter,
		cgroupPath:       cgroupPath,
		os:               osinterface.RealOS{},
		remoteCRI:        remoteCRI,
	}
	if enableNetworkPersistence {
		netPlugin, err := ocicni.InitCNI(networkPluginConfDir, networkPluginBinDir)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize cni plugin: %v", err)
		}
		c.netPlugin = netPlugin
		err = network.EnsureIPTableRules("cni0")
		if err != nil {
			return nil, fmt.Errorf("failed to apply iptable rules: %v", err)
		}
	}
	// prepare streaming server
	c.streamServer, err = NewStreamServer(criAdapter, c.containerStore, streamAddress, streamPort)
	if err != nil {
		return nil, fmt.Errorf("failed to create stream server: %v", err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create event monitor: %v", err)
	}
	// Create the grpc server and register runtime and image services.
	c.server = grpc.NewServer()
	babelFishRuntime := NewBabelFishRuntime(c)
	runtimeapi.RegisterImageServiceServer(c.server, babelFishRuntime)
	runtimeapi.RegisterRuntimeServiceServer(c.server, babelFishRuntime)

	return babelFishRuntime, nil
}

// Run starts the cri-babelfish service.
func (c *criBabelfishService) Run() error {
	klog.V(2).Info("Start cri-babelfish service")
	// Start event handler.
	klog.V(2).Info("Start event monitor")
	// Start streaming server.
	klog.V(2).Info("Start streaming server")
	streamServerCloseCh := make(chan struct{})
	go func() {
		if err := c.streamServer.Start(true); err != nil {
			klog.Errorf("Failed to start streaming server: %v", err)
		}
		close(streamServerCloseCh)
	}()

	// Start grpc server.
	// Unlink to cleanup the previous socket file.
	klog.V(2).Info("Start grpc server")
	err := syscall.Unlink(c.serverAddress)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to unlink socket file %q: %v", c.serverAddress, err)
	}
	l, err := net.Listen(unixProtocol, c.serverAddress)

	if err != nil {
		return fmt.Errorf("failed to listen on %q: %v", c.serverAddress, err)
	}
	grpcServerCloseCh := make(chan struct{})
	go func() {
		if err := c.server.Serve(l); err != nil {
			klog.Errorf("Failed to serve grpc grpc request: %v", err)
		}

		close(grpcServerCloseCh)
	}()

	// Stop the whole cri-babelfish service if any of the critical service exits.
	select {
	case <-streamServerCloseCh:
	case <-grpcServerCloseCh:
	}
	c.Stop()

	<-streamServerCloseCh
	klog.V(2).Info("Stream server stopped")
	<-grpcServerCloseCh
	klog.V(2).Info("GRPC server stopped")
	return nil
}

// Stop stops the cri-babelfish service.
func (c *criBabelfishService) Stop() {
	klog.V(2).Info("Stop cri-babelfish service")
	c.streamServer.Stop() // nolint: errcheck
	c.server.Stop()
}

func (c *criBabelfishService) GetContainer(sandBoxId string, containerID string) (*store.ContainerMetadata, error) {

	klog.V(4).Infof("Getting status of sandbox with ID in Service%s", sandBoxId)
	out := c.containerStore.List(containerID, sandBoxId)
	if len(out) != 0 {
		return out[0], nil
	} else {
		return nil, fmt.Errorf("Sandbox not found when getting its status")
	}
}
