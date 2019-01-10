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
	"multi-cri/pkg/cri/adapters/slurm/builder"
	"multi-cri/pkg/cri/adapters/slurm/cmd"
	"fmt"

	"multi-cri/pkg/cri/store"

	"strconv"

	"multi-cri/pkg/cri/common"
	"strings"

	"path/filepath"

	"multi-cri/pkg/cri/adapters"

	"k8s.io/klog"
	runtimeApi "k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2"
)

func (s SlurmAdapter) CreateContainer(cm *store.ContainerMetadata) error {
	klog.Infof("Creating container path in server")
	//mount parallel filesystem volume
	mounts := make(map[string]string)
	for _, m := range cm.Config.Mounts {
		mounts[m.ContainerPath] = m.HostPath
	}

	var err error
	mountPoint := s.MountPath
	if val, ok := mounts[adapters.VolumeContainer]; ok {
		cm.Extra["VolumePath"] = val
		cm.Extra["LocalPath"] = fmt.Sprintf("%s/%s/%s", val, cm.PodSandbox.ID, cm.ID)
		volName := filepath.Base(val)
		mountPoint = fmt.Sprintf("%s/%s", mountPoint, volName)
		logString(cm.LogFile, fmt.Sprintf("---\nContainer mounted in \"%s\" Volume.\nResults stored in directory:  \"%s/%s\" \n---",
			volName, cm.PodSandbox.ID, cm.ID))
	}
	cm.Extra["RMVolumePath"] = mountPoint
	cm.Extra["RMPath"] = fmt.Sprintf("%s/%s/%s", mountPoint, cm.PodSandbox.ID, cm.ID)

	//Ensure container path exists in SLURM cluster
	ensureRMPathExists(cm)

	//Pull image in SLURM cluster
	if err := s.Builder.PullImageInCluster(cm); err != nil {
		return err
	}

	klog.Infof("Created container path in server with id %s", cm.ID)
	return err
}

func (s SlurmAdapter) StartContainer(cm *store.ContainerMetadata) error {
	slurmClient, err := cmd.CreateCMD(cm)
	if err != nil {
		return err
	}

	// Create run singularity command
	jobConf := s.buildStartCommand(cm)

	//Filter system environment varaiables, so only container variables are set
	jobConf.ENV = filterEnvironmentVariables(cm)

	//Batch Job headers
	setupBatchHeaders(cm, jobConf)

	jobId, err := slurmClient.Sbatch(jobConf)
	if err != nil {
		return err
	}
	pid := 0
	if jobId != "" {
		pid, err = strconv.Atoi(jobId)
	}
	cm.Pid = pid
	return err
}

func setupBatchHeaders(cm *store.ContainerMetadata, jobConf *cmd.JobConfig) {
	jobConf.Headers = append(jobConf.Headers, cmd.JobConfigField{"-J", cm.Name})
	jobConf.Headers = append(jobConf.Headers, cmd.JobConfigField{"-o", StdoutFile})
	jobConf.Headers = append(jobConf.Headers, cmd.JobConfigField{"-e", SterrFile})
	if c, ok := cm.Environment["JOB_QUEUE"]; ok {
		jobConf.Headers = append(jobConf.Headers, cmd.JobConfigField{"-p", c})
	}
	if c, ok := cm.Environment["JOB_GPU"]; ok {
		jobConf.Headers = append(jobConf.Headers,
			cmd.JobConfigField{fmt.Sprintf("--gres=%s", c), ""})
	}
	if c, ok := cm.Environment["JOB_NUM_NODES"]; ok {
		jobConf.Headers = append(jobConf.Headers, cmd.JobConfigField{"-N", c})
	}
	if c, ok := cm.Environment["JOB_NUM_CORES_NODE"]; ok {
		jobConf.Headers = append(jobConf.Headers, cmd.JobConfigField{"-c", c})
	}
	if c, ok := cm.Environment["JOB_NUM_CORES"]; ok {
		jobConf.Headers = append(jobConf.Headers, cmd.JobConfigField{"-n", c})
	}
	if c, ok := cm.Environment["JOB_NUM_TASKS_NODE"]; ok {
		jobConf.Headers = append(jobConf.Headers,
			cmd.JobConfigField{fmt.Sprintf("--ntasks-per-node=%s", c), ""})
	}
	if c, ok := cm.Environment["JOB_CUSTOM_CONFIG"]; ok {
		jobConf.CustomHeaders = c
	}
}

func (s SlurmAdapter) StopContainer(cm *store.ContainerMetadata) error {
	slurmClient, err := cmd.CreateCMD(cm)
	if err != nil {
		return err
	}

	jobRef := cmd.JobReference{JobId: int32(cm.Pid)}
	return slurmClient.Scancel(jobRef)
}

func (s SlurmAdapter) ContainerStatus(cm *store.ContainerMetadata) error {
	slurmClient, err := cmd.CreateCMD(cm)
	if err != nil {
		return err
	}

	if cm.Pid != 0 {
		jobRef := &cmd.JobReference{JobId: int32(cm.Pid)}
		status, err := slurmClient.Sstatus(jobRef)
		if err != nil {
			return err
		}

		if status.JobState == "RUNNING" {
			cm.State = runtimeApi.ContainerState_CONTAINER_RUNNING
		} else if status.JobState == "COMPLETED" || status.JobState == "COMPLETING" {
			cm.State = runtimeApi.ContainerState_CONTAINER_EXITED
			cm.ExitCode = status.ExitCode
			cm.Reason = "completed"
			cm.FinishedAt = status.EndTime
		} else if status.JobState == "CANCELLED" || status.JobState == "TIMEOUT" {
			cm.State = runtimeApi.ContainerState_CONTAINER_EXITED
			cm.ExitCode = 1
			cm.Reason = "OOMKilled"
			cm.FinishedAt = status.EndTime
		} else if status.JobState == "FAILED" || status.JobState == "NODE_FAIL" {
			cm.State = runtimeApi.ContainerState_CONTAINER_EXITED
			if status.ExitCode == 0 {
				cm.ExitCode = 1
			}
			cm.Reason = status.JobState
		} else if status.JobState == "FAILED" || status.JobState == "NODE_FAIL" {
			cm.State = runtimeApi.ContainerState_CONTAINER_EXITED
			if status.ExitCode == 0 {
				cm.ExitCode = 1
			} else {
				cm.ExitCode = status.ExitCode
			}
			cm.Reason = status.JobState
			cm.Reason = "ContainerCannotRun"
			cm.FinishedAt = status.StarTime
		} else {
			cm.State = runtimeApi.ContainerState_CONTAINER_CREATED
		}
		if cm.State == runtimeApi.ContainerState_CONTAINER_EXITED {
			RMStderrPath := getRMStderrPath(cm.Extra["RMPath"])
			slurmClient.GetStderr(RMStderrPath)
			RMStdoutPath := getRMStdoutPath(cm.Extra["RMPath"])
			slurmClient.GetStdout(RMStdoutPath)
		}
	}

	return nil
}

func (s SlurmAdapter) ReopenContainerLog(cm *store.ContainerMetadata) error {
	return fmt.Errorf("SLURMCRU: ReopenContainerLog not implemented")
}

func (s SlurmAdapter) UpdateContainerResources(cm *store.ContainerMetadata) error {
	return fmt.Errorf("SLURMCRU: UpdateContainerResources not implemented")
}

func (s SlurmAdapter) buildStartCommand(c *store.ContainerMetadata) *cmd.JobConfig {
	var command string

	RMScriptPath := getRMScriptPath(c.Extra["RMPath"])
	//MPI job
	if _, ok := c.Environment["MPI_VERSION"]; ok {
		command = "mpirun"
		if flags, ok := c.Environment["MPI_FLAGS"]; ok {
			command = fmt.Sprintf("%s %s ", command, flags)
		}
	}
	command = fmt.Sprintf("%ssingularity exec %s", command, builder.GetRMImagePath(c, s.MountPath, s.ImageRemoteMount))

	for _, s := range c.Command {
		s = common.EscapeSpeciaCharacters(s)
		command = fmt.Sprintf("%s %s", command, s)
	}

	jobConf := &cmd.JobConfig{
		Command: command,
		Path:    c.Extra["RMPath"],
		Script:  RMScriptPath,
	}

	if c, ok := c.Environment["CLUSTER_CONFIG"]; ok {
		jobConf.Prerun = c
	}

	return jobConf
}

func filterEnvironmentVariables(c *store.ContainerMetadata) map[string]string {
	jobEnv := make(map[string]string)
	for k, v := range c.Environment {
		if strings.HasPrefix(k, "KUBERNETES_") {
			continue
		}
		if strings.HasPrefix(k, "CLUSTER_") {
			continue
		}
		if strings.HasPrefix(k, "JOB_") {
			continue
		}
		if strings.EqualFold(k, "MPI_FLAGS") {
			continue
		}
		jobEnv[k] = v
	}
	return jobEnv
}

/*
Execute command in remote via ssh
String: command
*/
func logString(logPath, message string) {
	stdoutWC, stderrWC, err := common.CreateContainerLoggers(logPath, false, 0)
	if err != nil {
		fmt.Printf("failed to start container logger: %s", err)
	}

	defer func() {
		stderrWC.Close()
		stdoutWC.Close()
	}()
	m := []byte(message)
	stdoutWC.Write(m)
}

func ensureRMPathExists(cm *store.ContainerMetadata) error {
	cli, err := cmd.CreateCMD(cm)
	if err != nil {
		return err
	}
	cmd := fmt.Sprintf("mkdir -p %s", cm.Extra["RMPath"])
	if _, err := cli.ExecCmd(cmd); err != nil {
		return err
	}
	return nil
}
