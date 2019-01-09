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

package cmd

import (
	"flag"
	"os/user"

	"github.com/spf13/pflag"
)

const defaultUnixSock = "/var/run/cri-babelfish.sock"

// CRIBabelfishOptions contains cri-babelfish command line options.
type CRIBabelFishOptions struct {
	//Adapter Name
	AdapterName string
	// SocketPath is the path to the socket which cri-babelfish serves on.
	SocketPath string
	// PrintVersion indicates to print version information of cri-babelfish.
	PrintVersion bool
	// CRIEndpoint is the babelfish endpoint path.
	CRIEndpoint string
	//Enable Pod Network
	EnablePodNetwork bool
	// NetworkPluginBinDir is the directory in which the binaries for the plugin is kept.
	NetworkPluginBinDir string
	// NetworkPluginConfDir is the directory in which the admin places a CNI conf.
	NetworkPluginConfDir string
	// StreamServerAddress is the ip address streaming server is listening on.
	StreamServerAddress string
	// StreamServerPort is the port streaming server is listening on.
	StreamServerPort string
	// CgroupPath is the path for the cgroup that cri-babelfish is placed in.
	CgroupPath string
	// EnableSelinux indicates to enable the selinux support
	EnableSelinux bool
	// SandboxImage is the image used by sandbox container.
	SandboxImage string
	//ResourceCachePath indicates the directory to store the CRI cache
	ResourceCachePath string
	//Enable Pod Persistence
	EnablePodPersistence bool
	//Remote CRI endpoints
	RemoteRuntime string
	//Image Remote Mount
	ImageRemoteMountPath string
}

// NewCRIBabelfishOptions
func NewCRIBabelFishOptions() *CRIBabelFishOptions {
	return &CRIBabelFishOptions{}
}

// AddFlags adds cri-babelfish command line options to pflag.
func (c *CRIBabelFishOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.AdapterName, "adapter-name",
		"fale", "Adapter name. For instance, \"slurm\" is supported")
	fs.StringVar(&c.SocketPath, "socket-path",
		defaultUnixSock, "Path to the socket which cri-babelfish serves on.")
	fs.BoolVar(&c.EnablePodNetwork, "enable-pod-network", false,
		"Enable pod network namespace")
	fs.StringVar(&c.NetworkPluginBinDir, "network-bin-dir",
		"/opt/cni/bin", "The directory for putting network binaries.")
	fs.StringVar(&c.NetworkPluginConfDir, "network-conf-dir",
		"/etc/cni/net.d", "The directory for putting network plugin configuration files.")
	fs.StringVar(&c.StreamServerAddress, "stream-addr",
		"", "The ip address streaming server is listening on. Default host interface is used if this is empty.")
	fs.StringVar(&c.StreamServerPort, "stream-port",
		"10010", "The port streaming server is listening on.")
	fs.StringVar(&c.SandboxImage, "sandbox-image",
		"gcr.io/google_containers/pause:3.0", "The image used by sandbox container.")
	fs.StringVar(&c.ResourceCachePath, "resources-cache-path", getHomeDir()+"/.cri-babelfish/",
		"Path where image, container and sandbox information will be stored. It will also be the image pool path")
	fs.BoolVar(&c.EnablePodPersistence, "enable-pod-persistence", false,
		"Enable pod and container persistence in cache file")
	fs.StringVar(&c.RemoteRuntime, "remote-runtime-endpoints", "default:/var/run/dockershim.sock",
		"Remote runtime endpoints to support RuntimeClass. Add several by separating them with comma")
}

// InitFlags must be called after adding all cli options flags are defined and
// before flags are accessed by the program. Ths function adds flag.CommandLine
// (the default set of command-line flags, parsed from os.Args) and then calls
// pflag.Parse().
func InitFlags() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
}

func getHomeDir() string {
	usr, _ := user.Current()
	if usr != nil {
		return usr.HomeDir
	}
	return "/root"
}
