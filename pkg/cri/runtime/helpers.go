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
	"os"
	"path/filepath"
	"strings"

	"multi-cri/pkg/cri/store"

	"github.com/containerd/cgroups"
	"github.com/cri-o/ocicni/pkg/ocicni"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"golang.org/x/net/context"
	runtime "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

const (
	maxDNSSearches = 6
	netNsRootDir   = "/etc/netns"
)

//load Cgroup path
func loadCgroup(cgroupPath string) (cgroups.Cgroup, error) {
	cg, err := cgroups.Load(cgroups.V1, cgroups.StaticPath(cgroupPath))
	if err != nil {
		if err != cgroups.ErrCgroupDeleted {
			return nil, err
		}
		if cg, err = cgroups.New(cgroups.V1, cgroups.StaticPath(cgroupPath), &specs.LinuxResources{}); err != nil {
			return nil, err
		}
	}
	if err := cg.Add(cgroups.Process{
		Pid: os.Getpid(),
	}); err != nil {
		return nil, err
	}
	return cg, nil
}

// toCNIPortMappings converts CRI port mappings to CNI.
func toCNIPortMappings(criPortMappings []*runtime.PortMapping) []ocicni.PortMapping {
	var portMappings []ocicni.PortMapping
	for _, mapping := range criPortMappings {
		if mapping.HostPort <= 0 {
			continue
		}
		portMappings = append(portMappings, ocicni.PortMapping{
			HostPort:      mapping.HostPort,
			ContainerPort: mapping.ContainerPort,
			Protocol:      strings.ToLower(mapping.Protocol.String()),
			HostIP:        mapping.HostIp,
		})
	}
	return portMappings
}

// parseDNSOptions parse DNS options into resolv.conf format content,
// if none option is specified, will return empty with no error.
func parseDNSOptions(servers, searches, options []string) (string, error) {
	resolvContent := ""

	if len(searches) > maxDNSSearches {
		return "", fmt.Errorf("DNSOption.Searches has more than 6 domains")
	}

	if len(searches) > 0 {
		resolvContent += fmt.Sprintf("search %s\n", strings.Join(searches, " "))
	}

	if len(servers) > 0 {
		resolvContent += fmt.Sprintf("nameserver %s\n", strings.Join(servers, "\nnameserver "))
	}

	if len(options) > 0 {
		resolvContent += fmt.Sprintf("options %s\n", strings.Join(options, " "))
	}

	return resolvContent, nil
}

// ensureImageExists returns corresponding metadata of the image reference, if image is not
// pulled yet, the function will pull the image.
func (c *MulticriRuntime) ensureImageExists(ctx context.Context, ref string) (*store.ImageMetadata, error) {
	image, err := c.imageStore.GetByPath(ref) //todo(jorgesece): remove, it is not used/tested
	/*if _, err := os.Stat("/path/to/whatever"); !os.IsNotExist(err) {
		return image, nil
	}*/
	if err != nil {
		return nil, fmt.Errorf("failed to resolve image %q: %v", ref, err)
	}
	if image != nil {
		return image, nil
	}
	// Pull image to ensure the image exists
	resp, err := c.PullImage(ctx, &runtime.PullImageRequest{Image: &runtime.ImageSpec{Image: ref}})
	if err != nil {
		return nil, fmt.Errorf("failed to pull image %q: %v", ref, err)
	}
	imageID := resp.GetImageRef()
	newImage, err := c.imageStore.Get(imageID)
	if err != nil {
		// It's still possible that someone removed the image right after it is pulled.
		return nil, fmt.Errorf("failed to get image %q metadata after pulling: %v", imageID, err)
	}
	return newImage, nil
}

// getSandboxRootDir returns the root directory for managing sandbox files,
// e.g. named pipes.
func getNetNsDir(netNsPath string) string {
	pathSplit := strings.Split(netNsPath, "/")
	return filepath.Join(netNsRootDir, pathSplit[len(pathSplit)-1])
}

// getSandboxRootDir returns the root directory for managing sandbox files,
// e.g. named pipes.
func getSandboxRootDir(rootDir, id string) string {
	return filepath.Join(sandboxesDir, id)
}

// getSandboxHosts returns the hosts file path inside the sandbox root directory.
func getSandboxHosts(sandboxRootDir string) string {
	return filepath.Join(sandboxRootDir, "hosts")
}

// getResolvPath returns resolv.conf filepath for specified sandbox.
func getResolvPath(sandboxRoot string) string {
	return filepath.Join(sandboxRoot, "resolv.conf")
}

// getSandboxDevShm returns the shm file path inside the sandbox root directory.
func getSandboxDevShm(sandboxRootDir string) string {
	return filepath.Join(sandboxRootDir, "shm")
}
