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
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cri-babelfish/pkg/cri/common"
	"cri-babelfish/pkg/cri/common/file"
	"cri-babelfish/pkg/cri/store"

	uuid "github.com/satori/go.uuid"
)

func recoverEnv(t *testing.T) {
	if r := recover(); r != nil {
		if strings.Contains(r.(string), "env variable is required") {
			t.Skip(r)
		}
		t.Errorf("test error")
	}
	t.Errorf("test error")
}

func createStructures(user, password, hostname, port string) (*store.SandboxMetadata, *store.ContainerMetadata, *store.ImageMetadata) {

	sandboxId := uuid.NewV4().String()
	sandbox := &store.SandboxMetadata{
		ID:      sandboxId,
		LogPath: file.GenerateLogDir(fmt.Sprintf("/tmp/%s", sandboxId), ""),
	}
	img := &store.ImageMetadata{
		RemotePath: "docker://alpine:latest",
		PodSandbox: *sandbox,
		LocalPath:  "/tmp/images",
		RepoType:   store.DockerRepositoryImageRepo,
	}
	containerEnv := make(map[string]string)
	containerEnv["CLUSTER_USERNAME"] = user
	containerEnv["CLUSTER_PASSWORD"] = password
	containerEnv["CLUSTER_HOST"] = hostname
	containerEnv["CLUSTER_PORT"] = port
	containerID := uuid.NewV4().String()
	c := &store.ContainerMetadata{
		ID:          containerID,
		PodSandbox:  *sandbox,
		Name:        "cri-slurm-test",
		Command:     []string{"sleep", "10"},
		LogFile:     store.GenerateContainerLogPath(sandbox.LogPath, containerID),
		Environment: containerEnv,
		Extra:       make(map[string]string),
	}
	os.MkdirAll(sandbox.LogPath, os.ModePerm)
	return sandbox, c, img

}

func TestIntegrationCreateStartContainer_integration(t *testing.T) {
	defer recoverEnv(t)
	password := common.GetEnv("TEST_SSH_PASSWORD", nil)
	user := common.GetEnv("TEST_SSH_USER", nil)
	host := common.GetEnv("TEST_SSH_HOST", nil)
	port := common.GetEnv("TEST_SSH_PORT", nil)
	_, c, img := createStructures(user, password, host, port)
	t.Logf("Check container logs in %s ", c.LogFile)
	if err := os.Setenv("CRI_SLURM_BUILD_IN_CLUSTER", "true"); err != nil {
		t.Fatal(err)
	}
	adap, err := NewSlurmAdapter()
	if err != nil {
		t.Error(err)
	}
	if err := adap.PullImage(img); err != nil {
		t.Fatal(err)
	}

	c.Image = img

	if err := adap.CreateContainer(c); err != nil {
		t.Fatal(err)
	}

	if err := adap.StartContainer(c); err != nil {
		t.Fatal(err)
	}
	timeout := time.After(20 * time.Second)
	tick := time.Tick(1 * time.Second)
	// Keep trying until we're timed out or got a result or got an error
	for {
		select {
		// Got a timeout! fail with a timeout error
		case <-timeout:
			t.Fatal("timed out")
			// Got a tick, we should check on doSomething()
		case <-tick:
			if err := adap.ContainerStatus(c); err != nil {
				t.Fatal(err)
			}
			if c.State > 1 {
				return
			}
		}
	}

	if c.State != 2 {
		t.Fatal("Container must be exited")
	}

}

func TestIntegrationCreateStartStopContainer_integration(t *testing.T) {
	defer recoverEnv(t)
	password := common.GetEnv("TEST_SSH_PASSWORD", nil)
	user := common.GetEnv("TEST_SSH_USER", nil)
	host := common.GetEnv("TEST_SSH_HOST", nil)
	port := common.GetEnv("TEST_SSH_PORT", nil)

	_, c, img := createStructures(user, password, host, port)
	t.Logf("Check container logs in %s ", c.LogFile)
	if err := os.Setenv("CRI_SLURM_BUILD_IN_CLUSTER", "true"); err != nil {
		t.Fatal(err)
	}
	adap, err := NewSlurmAdapter()
	if err != nil {
		t.Error(err)
	}

	if err := adap.PullImage(img); err != nil {
		t.Fatal(err)
	}
	c.Image = img

	if err := adap.CreateContainer(c); err != nil {
		t.Fatal(err)
	}
	if err := adap.StartContainer(c); err != nil {
		t.Fatal(err)
	}
	if err := adap.StopContainer(c); err != nil {
		t.Fatal(err)
	}
	if err := adap.ContainerStatus(c); err != nil {
		t.Fatal(err)
	}
	if c.State != 2 {
		t.Fatal("Container must be cancelled")
	}

}
