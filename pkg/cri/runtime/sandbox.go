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
	"multi-cri/pkg/cri/common/file"
	"fmt"

	"multi-cri/pkg/cri/store"

	"github.com/cri-o/ocicni/pkg/ocicni"
	"golang.org/x/net/context"
	"k8s.io/klog"
	runtimeApi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

const (
	// sandboxesDir contains all sandbox root. A sandbox root is the running
	// directory of the sandbox, all files created for the sandbox will be
	// placed under this directory.
	sandboxesDir = "sandboxes"
	// etcHosts is the default path of /etc/hosts file.
	etcHosts = "/etc/hosts"
	// resolvConfPath is the abs path of resolv.conf on host or container.
	resolvConfPath = "/etc/resolv.conf"
)

func (r *MulticriRuntime) RunPodSandbox(ctx context.Context, req *runtimeApi.RunPodSandboxRequest) (_ *runtimeApi.RunPodSandboxResponse, retErr error) {
	klog.V(4).Infof("Creating Multicri sandbox")
	var err error
	config := req.GetConfig()

	//Remote runtime. RuntimeHandler
	response, err := r.remoteCRI.RunPodSandbox(ctx, req)
	if err != nil {
		return nil, err
	}
	// Create initial internal sandbox object.

	if response != nil {
		config.Metadata.Uid = response.PodSandboxId
	}
	sandbox := r.sandboxStore.CreateSandboxMetadata(runtimeApi.PodSandboxState_SANDBOX_NOTREADY, *config, req.RuntimeHandler)

	if response == nil {

		defer func() {
			// Release the name if the function returns with an error.
			if retErr != nil {
				r.sandboxStore.Remove(sandbox.ID)
			}
		}()
		if r.netPlugin != nil {
			if err := r.setupPodNetwork(sandbox); err != nil {
				return nil, err
			}
		}

		errPath := file.GenerateLogDir(req.Config.LogDirectory, sandbox.ID)
		err = file.CreatePathDirectory(errPath)
		if err != nil {
			klog.Errorf("Error when creating the log directory %s.", req.Config.LogDirectory)
		}
		sandbox.LogPath = errPath
		if err := r.adapter.RunPodSandbox(sandbox); err != nil {
			return nil, err
		}
		response = &runtimeApi.RunPodSandboxResponse{PodSandboxId: sandbox.ID}
	}

	r.sandboxStore.Update(sandbox)
	klog.V(4).Infof("Multicri sandbox successfully created")
	return response, nil
}

func (r *MulticriRuntime) StopPodSandbox(ctx context.Context, req *runtimeApi.StopPodSandboxRequest) (*runtimeApi.StopPodSandboxResponse, error) {
	sandbox, errGet := r.sandboxStore.Get(req.GetPodSandboxId())
	klog.V(4).Infof("Stopping sandbox with ID %s", req.GetPodSandboxId())
	if errGet != nil {
		sandbox = &store.SandboxMetadata{}
	}
	response, err := r.remoteCRI.StopPodSandbox(sandbox.RuntimeHandler, ctx, req)
	if err != nil {
		return nil, err
	}
	if response == nil {
		if errGet != nil {
			return nil, fmt.Errorf("Sandbox not found when stopping it")
		}
		if sandbox.NetNSPath != "" {
			if err := r.tearDownNetwork(sandbox); err != nil {
				return nil, err
			}
		}
		if err := r.adapter.StopPodSandbox(sandbox); err != nil {
			return nil, err
		}
		response = &runtimeApi.StopPodSandboxResponse{}
	}
	if errGet == nil {
		sandbox.State = runtimeApi.PodSandboxState_SANDBOX_NOTREADY
		sandbox.IP = ""
		r.sandboxStore.Update(sandbox)
	}
	klog.V(4).Infof("Multicri sandbox successfully stopped")
	return response, nil
}

func (r *MulticriRuntime) RemovePodSandbox(ctx context.Context, req *runtimeApi.RemovePodSandboxRequest) (*runtimeApi.RemovePodSandboxResponse, error) {
	sandbox, errGet := r.sandboxStore.Get(req.GetPodSandboxId())
	klog.V(4).Infof("Removing sandbox with ID %s", req.GetPodSandboxId())
	if errGet != nil {
		sandbox = &store.SandboxMetadata{}
	}

	response, err := r.remoteCRI.RemovePodSandbox(sandbox.RuntimeHandler, ctx, req)
	if err != nil {
		return nil, err
	}
	if response == nil {
		if errGet != nil {
			return nil, fmt.Errorf("Sandbox not found when removing it")
		}
		if sandbox.State == runtimeApi.PodSandboxState_SANDBOX_READY {
			return nil, fmt.Errorf("Sandbox %q is running. First stop it ", req.GetPodSandboxId())
			//todo(jorgesece): manage force remove
		}
		if err := r.adapter.RemovePodSandbox(sandbox); err != nil {
			return nil, err
		}
		response = &runtimeApi.RemovePodSandboxResponse{}
	}

	r.sandboxStore.Remove(sandbox.ID)
	klog.V(4).Infof("Multicri sandbox successfully removed")
	return response, nil
}

func (r *MulticriRuntime) PodSandboxStatus(ctx context.Context, req *runtimeApi.PodSandboxStatusRequest) (*runtimeApi.PodSandboxStatusResponse, error) {
	sandbox, err := r.sandboxStore.Get(req.GetPodSandboxId())
	klog.V(4).Infof("Getting status of sandbox with ID %s", req.GetPodSandboxId())
	if err != nil {
		return &runtimeApi.PodSandboxStatusResponse{},
			fmt.Errorf("Sandbox not found when getting its status %s", err)
	}
	//RuntimeHandler for remote cris
	response, err := r.remoteCRI.PodSandboxStatus(sandbox.RuntimeHandler, ctx, req)
	if err != nil {
		return nil, err
	}
	if response == nil {

		ip := sandbox.IP
		if ip != "" {
			sandbox.State = runtimeApi.PodSandboxState_SANDBOX_READY
		} else if r.netPlugin == nil {
			ip = "127.0.0.1"
			sandbox.State = runtimeApi.PodSandboxState_SANDBOX_READY
		}
		if err := r.adapter.PodSandboxStatus(sandbox); err != nil {
			return nil, err
		}
		status := store.ParseToK8sSandboxStatus(sandbox, ip)
		response = &runtimeApi.PodSandboxStatusResponse{Status: status}
	}
	return response, err
}

func (r *MulticriRuntime) ListPodSandbox(ctx context.Context, req *runtimeApi.ListPodSandboxRequest) (*runtimeApi.ListPodSandboxResponse, error) {
	klog.V(4).Infof("Listing sandboxes")
	localResult := r.sandboxStore.ListK8s(req.Filter, r.remoteCRI.MulticriRuntimeName())
	remotePods, err := r.remoteCRI.ListPodSandbox(ctx, req)
	if err != nil {
		return nil, err
	}
	localResult = append(localResult, remotePods...)

	return &runtimeApi.ListPodSandboxResponse{Items: localResult}, nil
}

func (r *MulticriRuntime) PortForward(ctx context.Context, req *runtimeApi.PortForwardRequest) (*runtimeApi.PortForwardResponse, error) {
	sandbox, errGet := r.sandboxStore.Get(req.GetPodSandboxId())
	klog.V(4).Infof("Forwarding port of sandbox %s", req.GetPodSandboxId())
	var runtimeClass string
	if errGet == nil {
		runtimeClass = sandbox.RuntimeHandler
	}
	//RuntimeHandler for remote cris
	response, err := r.remoteCRI.PortForward(runtimeClass, ctx, req)
	if err != nil {
		return nil, err
	}
	if response != nil {
		return response, err
	}
	if errGet != nil {
		return nil, fmt.Errorf("Sandbox not found when portforwarding it")
	}
	return r.streamServer.GetPortForward(req)
}

// setupNetSandboxFiles sets up necessary network sandbox files including /etc/hosts and /etc/resolv.conf.
func (r *MulticriRuntime) setupNetSandboxFiles(netDir string, config *runtimeApi.PodSandboxConfig) error {
	// TODO(random-liu): Consider whether we should maintain /etc/hosts and /etc/resolv.conf in kubelet.
	sandboxEtcHosts := getSandboxHosts(netDir)
	if err := r.os.CopyFile(etcHosts, sandboxEtcHosts, 0644); err != nil {
		return fmt.Errorf("failed to generate sandbox hosts file %q: %v", sandboxEtcHosts, err)
	}

	// Set DNS options. Maintain a resolv.conf for the sandbox.
	var err error
	resolvContent := ""
	if dnsConfig := config.GetDnsConfig(); dnsConfig != nil {
		resolvContent, err = parseDNSOptions(dnsConfig.Servers, dnsConfig.Searches, dnsConfig.Options)
		if err != nil {
			return fmt.Errorf("failed to parse sandbox DNSConfig %+v: %v", dnsConfig, err)
		}
	}
	resolvPath := getResolvPath(netDir)
	if resolvContent == "" {
		// copy host's resolv.conf to resolvPath
		err = r.os.CopyFile(resolvConfPath, resolvPath, 0644)
		if err != nil {
			return fmt.Errorf("failed to copy host's resolv.conf to %q: %v", resolvPath, err)
		}
	} else {
		err = r.os.WriteFile(resolvPath, []byte(resolvContent), 0644)
		if err != nil {
			return fmt.Errorf("failed to write resolv content to %q: %v", resolvPath, err)
		}
	}

	return nil
}

func (r *MulticriRuntime) setupPodNetwork(sandbox *store.SandboxMetadata) (retErr error) {
	//Create Network Namespace if it is not in host network
	hostNet := sandbox.Config.GetLinux().GetSecurityContext().GetNamespaceOptions().GetNetwork()
	if hostNet == runtimeApi.NamespaceMode_POD {
		klog.V(4).Infof("Creating namespace...")
		// If it is not in host network namespace then create a namespace and set the sandbox
		// handle. NetNSPath in sandbox metadata and NetNS is non empty only for non host network
		// namespaces. If the pod is in host network namespace then both are empty and should not
		// be used.
		netNS, err := r.networkNamespace.OpenNetNamespace("")
		if err != nil {
			return fmt.Errorf("failed to create network namespace for sandbox: %v", err)

		}
		NetNSPath := netNS.GetPath()
		defer func() {
			if retErr != nil {
				if err := netNS.Remove(); err != nil {
					klog.Errorf("Failed to remove network namespace %s for sandbox %s: %v", NetNSPath, sandbox.ID, err)

				}
				NetNSPath = ""
			}
		}()
		// Setup network for sandbox.
		podNetwork := ocicni.PodNetwork{
			Name:         sandbox.Config.GetMetadata().GetName(),
			Namespace:    sandbox.Config.GetMetadata().GetNamespace(),
			ID:           sandbox.ID,
			NetNS:        sandbox.NetNSPath,
			PortMappings: toCNIPortMappings(sandbox.Config.GetPortMappings()),
		}
		if err = r.netPlugin.SetUpPod(podNetwork); err != nil {
			return fmt.Errorf("failed to setup network for sandbox %q: %v", sandbox.ID, err)
		}
		sandbox.IP, err = r.netPlugin.GetPodNetworkStatus(podNetwork)
		if err != nil {
			return fmt.Errorf("failed to get IP for sandbox %q: %v", sandbox.ID, err)
		}
		defer func() {
			if retErr != nil {
				// Teardown network if an error is returned.
				if err := r.netPlugin.TearDownPod(podNetwork); err != nil {
					klog.Errorf("Failed to destroy network for sandbox %q: %v", sandbox.ID, err)
				}
			}
		}()
	}
	//Set DNS
	sandboxNetNsDir := getNetNsDir(sandbox.NetNSPath)
	if err := r.os.MkdirAll(sandboxNetNsDir, 0755); err != nil {
		return fmt.Errorf("failed to create sandbox root directory %q: %v", sandboxNetNsDir, err)
	}
	defer func() {
		if retErr != nil {
			// Cleanup the sandbox root directory.
			if err := r.os.RemoveAll(sandboxNetNsDir); err != nil {
				klog.Errorf("Failed to remove sandbox root directory %q: %v",
					sandboxNetNsDir, err)
			}
		}
	}()

	// Setup sandbox /dev/shm, /etc/hosts and /etc/resolv.conf.
	if err := r.setupNetSandboxFiles(sandboxNetNsDir, &sandbox.Config); err != nil {
		return fmt.Errorf("failed to setup sandbox files: %v", err)
	}
	return nil
}

func (r *MulticriRuntime) tearDownNetwork(sandbox *store.SandboxMetadata) error {
	if _, err := r.os.Stat(sandbox.NetNSPath); err != nil {
		if !r.os.IsNotExist(err) {
			return fmt.Errorf("failed to stat network namespace path %s :%v", sandbox.NetNSPath, err)
		}
	} else {
		if teardownErr := r.netPlugin.TearDownPod(ocicni.PodNetwork{
			Name:         sandbox.Config.GetMetadata().GetName(),
			Namespace:    sandbox.Config.GetMetadata().GetNamespace(),
			ID:           sandbox.ID,
			NetNS:        sandbox.NetNSPath,
			PortMappings: toCNIPortMappings(sandbox.Config.GetPortMappings()),
		}); teardownErr != nil {
			return fmt.Errorf("failed to destroy network for sandbox %q: %v", sandbox.ID, teardownErr)
		}
	}

	//Close the sandbox network namespace if it was created
	netNS, err := r.networkNamespace.OpenNetNamespace(sandbox.NetNSPath)
	if err == nil {
		if err = netNS.Remove(); err != nil {
			return fmt.Errorf("failed to remove network namespace for sandbox %q:  %v", sandbox.ID, err)
		}
		sandboxNetNsDir := getNetNsDir(sandbox.NetNSPath)
		if err = r.os.RemoveAll(sandboxNetNsDir); err != nil {
			return fmt.Errorf("failed to remove network namespace dir for sandbox %q:  %v", sandbox.ID, err)
		}
	} else {
		klog.Errorf("Network namespace %s not found for sandbox %q:  %v", sandbox.NetNSPath, sandbox.ID, err)
	}
	return nil
}
