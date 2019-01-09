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

package main

import (
	"os"

	"cri-babelfish/pkg/cmd"
	"cri-babelfish/pkg/cri/runtime"

	"k8s.io/klog"
	"github.com/opencontainers/selinux/go-selinux"
	"github.com/spf13/pflag"
	"k8s.io/apiserver/pkg/util/flag"
	"k8s.io/apiserver/pkg/util/logs"
	"k8s.io/kubernetes/pkg/util/interrupt"
)

func main() {
	o := cmd.NewCRIBabelFishOptions()
	o.AddFlags(pflag.CommandLine)
	flag.InitFlags()
	logs.InitLogs()
	defer logs.FlushLogs()

	if !o.EnableSelinux {
		selinux.SetDisabled()
	}

	klog.V(2).Infof("Run cri-babelfish grpc server on socket %q", o.SocketPath)
	klog.Infof("Run cri-babelfish grpc server on socket")
	s, err := runtime.NewBabelFishService(
		o.AdapterName,
		o.SocketPath,
		o.NetworkPluginBinDir,
		o.NetworkPluginConfDir,
		o.StreamServerAddress,
		o.StreamServerPort,
		o.CgroupPath,
		o.SandboxImage,
		o.ResourceCachePath,
		o.EnablePodPersistence,
		o.EnablePodNetwork,
		o.RemoteRuntime,
	)

	if err != nil {
		klog.Exitf("Failed to create CRI babelfish service %+v: %v", o, err)
	}
	// Use interrupt handler to make sure the server to be stopped properly.
	h := interrupt.New(func(os.Signal) {}, s.Stop)
	if err := h.Run(func() error { return s.Run() }); err != nil {
		klog.Exitf("Failed to run cri-babelfish grpc server: %v", err)
	}
}
