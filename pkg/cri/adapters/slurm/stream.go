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

package slurm

import (
	"multi-cri/pkg/cri/store"
	"fmt"
	"io"

	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubernetes/pkg/kubelet/server/streaming"
)

type streamRuntime struct {
	c store.ContainerStoreInterface
}

func (r SlurmAdapter) NewStreamRuntime(c store.ContainerStoreInterface) streaming.Runtime {
	return &streamRuntime{c: c}
}

func (r *streamRuntime) Attach(containerID string, in io.Reader, out, err io.WriteCloser, tty bool,
	resize <-chan remotecommand.TerminalSize) error {
	return fmt.Errorf("SLURMCRI: streamRuntime Attach still not implemented")
}

func (r *streamRuntime) Exec(containerID string, cmd []string, stdin io.Reader, stdout, stderr io.WriteCloser,
	tty bool, resize <-chan remotecommand.TerminalSize) error {
	return fmt.Errorf("SLURMCRI: streamRuntime Exec still not implemented")

}

func (r *streamRuntime) PortForward(podSandboxID string, port int32, stream io.ReadWriteCloser) error {
	return fmt.Errorf("SLURMCRI: streamRuntime PortForward still not implemented")
}
