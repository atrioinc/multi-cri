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
	"fmt"
	"os"
	osexec "os/exec"
	"strings"

	"k8s.io/klog"
)

type SingularityCLI interface {
	RunAsyncCommand(command []string) error
	RunSyncCommand(command []string) ([]string, error)
	SingularityPullImage(imagePath string, remoteImage string, auth string) error
}

type CLIConfig struct {
	Debug   bool   `flag:"debug"`
	Version string `flag:"version"`
}

type cli struct {
	singularityPath string
	config          CLIConfig

	globalFlags []string
}

func NewSingularityCLI(singularityPath string, cfg CLIConfig) (SingularityCLI, error) {
	if singularityPath == "" {
		singularityP, err := osexec.LookPath("singularity")
		if err != nil {
			return nil, fmt.Errorf("must have Singularity 3.0 installed: %v", err)
		}
		singularityPath = singularityP
	}

	if _, err := os.Stat(singularityPath); err != nil {
		return nil, fmt.Errorf("singularity binary did not exist at %q: %v", singularityPath, err)
	}
	return &cli{singularityPath: singularityPath, config: cfg}, nil
}

func (c *cli) SingularityPullImage(imagePath string, imageRemote string, auth string) error {
	var args []string
	command, err := c.generateSingularityImageCommand("pull", imagePath, imageRemote, args, auth)
	if err != nil {
		return fmt.Errorf("Error building the singularity stop command")
	}
	err = c.RunAsyncCommand(command)
	if err != nil {
		return err
	}
	return nil
}

// RunCommand runs singularity command related to the container management
func (c *cli) RunAsyncCommand(command []string) error {
	var err error
	klog.V(4).Infof("singularity: calling cmd %v", command)
	// Create command
	cmd := osexec.Command(command[0], command[1:]...)
	//Output to system by default
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	//execute
	err = cmd.Run()
	if err != nil {
		klog.Warningf("singularity: cmd %v %v errored with %v", command, err)
		return fmt.Errorf("failed to run %v: %v", command, err)
	}

	return nil
}

// RunCommand runs singularity command related to the container management
func (c *cli) RunSyncCommand(command []string) ([]string, error) {
	klog.V(4).Infof("singularity: calling cmd %v", command)
	// Create command
	cmd := osexec.Command(command[0], command[1:]...)
	//execute
	out, err := cmd.CombinedOutput()
	if err != nil {
		klog.Warningf("singularity: cmd %v errored with %v", command, err)
		return strings.Split(strings.TrimSpace(string(out)), "\n"),
			fmt.Errorf("failed to run %v: %v", command, err)
	}

	return strings.Split(strings.TrimSpace(string(out)), "\n"), nil
}

//Generate singularity pull command
func (c *cli) generateSingularityImageCommand(subCommand string, image string, instanceName string, args []string, auth string) ([]string, error) {
	cmd_args := []string{}
	if len(args) > 0 {
		cmd_args = append(cmd_args, args...)
	}
	cmd_args = append(cmd_args, image)
	if instanceName != "" {
		cmd_args = append(cmd_args, instanceName)
	}

	cmd := []string{}
	if auth != "" {
		cmd = append(cmd, auth)
	}
	cmd = append(cmd, c.singularityPath, subCommand)
	cmd = append(append(cmd, c.globalFlags...), cmd_args...)
	return cmd, nil
}
