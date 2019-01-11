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
	"multi-cri/pkg/cri/common/file"
	"multi-cri/pkg/cri/common/ssh"
	"path"

	"os"

	"bytes"
	"multi-cri/pkg/cri/common"
	"fmt"
	"io"
	"strconv"
	"strings"

	"io/ioutil"

	"multi-cri/pkg/cri/store"

	"k8s.io/klog"
)

const (
	PreRunScript = "prerun.sh"
	BatchScript  = "batch.sh"
)

type JobConfigField struct {
	Flag  string
	Value string
}
type JobConfig struct {
	Headers       []JobConfigField
	CustomHeaders string
	Command       string
	Path          string
	Script        string
	Prerun        string
	ENV           map[string]string
}

type JobStatus struct {
	JobState string
	ExitCode int
	Reason   string
	StarTime int64
	EndTime  int64
}

type JobReference struct {
	JobId int32
}

type SlurmCmd struct {
	sshClient *ssh.SSH
	logPath   string
}

func CreateCMD(metadata *store.ContainerMetadata) (*SlurmCmd, error) {
	user := metadata.Environment["CLUSTER_USERNAME"]
	host := metadata.Environment["CLUSTER_HOST"]
	port := metadata.Environment["CLUSTER_PORT"]
	var password string
	var key string
	if p, ok := metadata.Environment["CLUSTER_PASSWORD"]; ok {
		password = p
	}
	if k, ok := metadata.Environment["CLUSTER_KEYVALUE"]; ok {
		key = k
	}
	return NewSlurmCMD(user, host, port, metadata.LogFile, key, password)
}

func NewSlurmCMD(user, host, port, logPath, key, password string) (*SlurmCmd, error) {
	var sshClient ssh.SSH
	if key != "" {
		k := []byte(key)
		sshClient = ssh.NewSSH(user, host, port, nil, nil, k)
	} else if password != "" {
		sshClient = ssh.NewSSH(user, host, port, nil, &password, nil)
	} else {
		return nil, fmt.Errorf("KeyPath or password must be setup")
	}

	sl := &SlurmCmd{
		sshClient: &sshClient,
		logPath:   logPath,
	}

	return sl, nil
}

/*
Execute command in remote via ssh
String: command
*/
func (s SlurmCmd) ExecCmd(cmd string) (string, error) {
	klog.V(4).Infof("Execute command %s", cmd)
	stdoutWC, stderrWC, err := common.CreateContainerLoggers(s.logPath, false, 100)
	if err != nil {
		return "", fmt.Errorf("failed to start container logger: %s", err)
	}

	defer func() {
		stderrWC.Close()
		stdoutWC.Close()
	}()

	response, err := s.run(cmd, stdoutWC, stderrWC)
	if err != nil {
		return "", fmt.Errorf("%s. %s", response, err)
	}
	return response, nil
}

/*
Get Stderr via ssh
String: stderrpath
*/
func (s SlurmCmd) GetStderr(stdoerrPath string) (string, error) {
	klog.V(4).Infof("Get Job Error%s", stdoerrPath)
	stdoutWC, stderrWC, err := common.CreateContainerLoggers(s.logPath, false, 100)
	if err != nil {
		return "", fmt.Errorf("failed to start container logger: %s", err)
	}

	defer func() {
		stderrWC.Close()
		stdoutWC.Close()
	}()
	cmd := fmt.Sprintf("cat %s", stdoerrPath)
	response, err := s.run(cmd, stderrWC, stderrWC)
	if err != nil {
		return "", fmt.Errorf("%s. %s", response, err)
	}
	return response, nil
}

/*
Get Stout via ssh
String: stdoutpath
*/
func (s SlurmCmd) GetStdout(stdoerrPath string) (string, error) {
	klog.V(4).Infof("Get Job output %s", stdoerrPath)
	cmd := fmt.Sprintf("cat %s", stdoerrPath)
	return s.ExecCmd(cmd)
}

func (s SlurmCmd) run(cmd string, stdout, stderr io.WriteCloser) (string, error) {
	response, _, err := s.sshClient.Run(cmd, stdout, stderr, true)
	return response, err
}

/*
Execute sbatch command in a slurm cluster.
Returns JobID
Returns error
*/
func (s SlurmCmd) Sbatch(config *JobConfig) (string, error) {
	klog.V(4).Infof("Execute batch %s", config.Script)
	err := s.batchScript(config)
	if err != nil {
		return "", err
	}
	//Pod logs
	stdoutWC, stderrWC, err := common.CreateContainerLoggers(s.logPath, false, 100)
	if err != nil {
		return "", fmt.Errorf("failed to start container logger: %s", err)
	}
	defer func() {
		stderrWC.Close()
		stdoutWC.Close()
	}()
	//run command
	response, err := s.run(config.Script, stdoutWC, stderrWC)
	if err != nil {
		return "", err
	}
	jobId, err := parseJobId(response)
	if err != nil {
		return "", err
	}
	return jobId, nil
}

/*
Copy local path to remote path
*/
func (s SlurmCmd) CopyTo(resourcePath, destinyPath string) error {
	err := s.sshClient.CopyTo(resourcePath, destinyPath)
	if err != nil {
		return fmt.Errorf("Error copying file to ssh server. %s ", err)
	}
	return nil
}

/*
Copy local path to local path
*/
func (s SlurmCmd) CopyInternal(resourcePath, destinyPath string) error {
	cmd := fmt.Sprintf("cp -r %s %s", resourcePath, destinyPath)
	_, err := s.ExecCmd(cmd)
	if err != nil {
		return fmt.Errorf("Error copying folder/file inside the server cluster. %s ", err)
	}
	return nil
}

/*
Cancel a specific job
*/
func (s SlurmCmd) Scancel(reference JobReference) error {
	klog.V(4).Infof("Canceling job %d", reference.JobId)
	stdoutWC, stderrWC, err := common.CreateContainerLoggers(s.logPath, false, 100)
	if err != nil {
		return fmt.Errorf("failed to start container logger: %s", err)
	}
	defer func() {
		stderrWC.Close()
		stdoutWC.Close()
	}()
	//build command
	cmd := fmt.Sprintf("scancel %d", reference.JobId)
	//run command
	_, err = s.run(cmd, stdoutWC, stderrWC)
	if err != nil {
		return err
	}
	return nil
}

/*
Get job status
*/
func (s SlurmCmd) Sstatus(reference *JobReference) (*JobStatus, error) {
	klog.V(4).Infof("Check status for job %d", reference.JobId)
	stdoutWC, stderrWC, err := common.CreateContainerLoggers(s.logPath, false, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to start container logger: %s", err)
	}
	defer func() {
		stderrWC.Close()
		stdoutWC.Close()
	}()
	out, err := s.scontrol(reference, stdoutWC, stderrWC)
	if err != nil {
		klog.V(5).Infof("scontrol command fails. %s", err)
		return s.sacct(reference, stdoutWC, stderrWC)
	}
	return out, err
}

func (s SlurmCmd) sacct(jobRef *JobReference, stdoutWC, stderrWC io.WriteCloser) (*JobStatus, error) {
	cmd := fmt.Sprintf("sacct -p -n -j %d -o start,end,exitcode,start,state,comment", jobRef.JobId)
	response, err := s.run(cmd, stdoutWC, stderrWC)
	if err != nil {
		return nil, fmt.Errorf("Retrieve job info fails %s ", err)
	}
	return parseAcctStatus(response)
}

func (s SlurmCmd) scontrol(jobRef *JobReference, stdoutWC, stderrWC io.WriteCloser) (*JobStatus, error) {
	//build command
	cmd := fmt.Sprintf("scontrol show jobid -dd  %d", jobRef.JobId)
	//run command
	response, err := s.run(cmd, stdoutWC, stderrWC)
	if err != nil {
		return nil, fmt.Errorf("Retrieve job info fails %s ", err)
	}
	return parseControlStatus(response)
}

func (s SlurmCmd) batchScript(config *JobConfig) error {
	//Pre run
	if config.Prerun != "" {
		prerun := fmt.Sprintf("%s/%s", config.Path, PreRunScript)
		prerunScript, err := buildPreRunScript(config.Prerun)
		if err != nil {
			return fmt.Errorf("Error generating batch script %s ", err)
		}
		err = s.CopyTo(prerunScript, prerun)
		if err != nil {
			return fmt.Errorf("Error copying prerun file to Slurm cluster. %s ", err)
		}
	}
	//Batch
	batchScript := fmt.Sprintf("%s/%s", config.Path, BatchScript)
	batchLocalPath, err := buildSbatch(config)
	if err != nil {
		return fmt.Errorf("Error generating batch script %s ", err)
	}
	err = s.CopyTo(batchLocalPath, batchScript)
	if err != nil {
		return fmt.Errorf("Error copying batch file to Slurm cluster. %s ", err)
	}

	//Run script
	runLocalScript, err := buildRunScript(config)
	if err != nil {
		return fmt.Errorf("Error generating run script %s ", err)
	}
	err = s.CopyTo(runLocalScript, config.Script)
	if err != nil {
		return fmt.Errorf("Error copying run file to Slurm cluster. %s ", err)
	}
	return nil
}

func buildPreRunScript(clusterConf string) (string, error) {
	var b bytes.Buffer

	if err := writeNewLine(&b, clusterConf); err != nil {
		return "", err
	}

	//Copy to file
	filePath := file.GenerateTmpFile("/tmp", "prerun", "sh")
	// open output file
	fo, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
	if err != nil {
		return "", err
	}
	if _, err := fo.Write(b.Bytes()); err != nil {
		return "", err
	}
	return filePath, nil
}
func buildRunScript(config *JobConfig) (string, error) {
	var b bytes.Buffer

	content := "#!/bin/bash"
	if err := writeNewLine(&b, content); err != nil {
		return "", err
	}

	//export env variables
	var envString string
	for key, value := range config.ENV {
		envString = fmt.Sprintf("%sexport %s=\"%s\" \n", envString, key, value)
	}
	if envString != "" {
		if err := writeNewLine(&b, envString); err != nil {
			return "", err
		}
	}
	////move to path, container result files are stored there
	path := fmt.Sprintf("cd %s", config.Path)
	if err := writeNewLine(&b, path); err != nil {
		return "", err
	}
	//source prerun
	if config.Prerun != "" {
		precmd := fmt.Sprintf("source %s", PreRunScript)
		if err := writeNewLine(&b, precmd); err != nil {
			return "", err
		}
	}

	//sbatch script
	cmd := fmt.Sprintf("sbatch %s", BatchScript)
	if err := writeNewLine(&b, cmd); err != nil {
		return "", err
	}

	//Copy to file
	filePath := file.GenerateTmpFile("/tmp", "run", "sh")
	// open output file
	fo, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
	if err != nil {
		return "", err
	}
	if _, err := fo.Write(b.Bytes()); err != nil {
		return "", err
	}

	return filePath, nil
}

func buildSbatch(config *JobConfig) (string, error) {
	var commands []string
	if len(config.Headers) == 0 {
		return "", fmt.Errorf("no configuration provided")
	}
	var b bytes.Buffer

	commands = append(commands, "#!/bin/bash")
	if config.CustomHeaders != "" {
		commands = append(commands, config.CustomHeaders)
	}
	for _, c := range config.Headers {
		content := fmt.Sprintf("#SBATCH %s %s", c.Flag, c.Value)
		commands = append(commands, content)
	}
	commands = append(commands, config.Command)

	if err := writeLines(&b, commands); err != nil {
		return "", err
	}

	filePath := file.GenerateTmpFile("/tmp", "batch", "sh")
	// open output file
	fo, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
	if err != nil {
		return "", err
	}
	if _, err := fo.Write(b.Bytes()); err != nil {
		return "", err
	}
	return filePath, nil
}

func writeNewLine(b *bytes.Buffer, content string) error {
	text := fmt.Sprintf("%s \n", content)
	if _, err := b.WriteString(text); err != nil {
		return err
	}
	return nil
}

func writeLines(b *bytes.Buffer, content []string) error {
	for _, v := range content {
		if err := writeNewLine(b, v); err != nil {
			return err
		}
	}
	return nil
}

func parseControlStatus(stdout string) (*JobStatus, error) {
	jobInfo := map[string]string{}
	lines := strings.Split(stdout, "\n")
	for _, l := range lines {
		items := strings.Split(strings.TrimSpace(l), " ")
		for _, i := range items {
			item := strings.Split(i, "=")
			if len(item) > 1 {
				jobInfo[strings.TrimSpace(item[0])] = strings.TrimSpace(item[1])
			}
		}
	}
	exitCode := 0
	if val, ok := jobInfo["ExitCode"]; ok {
		exitCode, _ = strconv.Atoi(val)
	}
	start := common.ParseDate(jobInfo["StartTime"])
	end := common.ParseDate(jobInfo["EndTime"])
	return &JobStatus{ExitCode: exitCode, JobState: jobInfo["JobState"],
		Reason: jobInfo["Reason"], EndTime: end, StarTime: start,
	}, nil
}

func parseAcctStatus(stdout string) (*JobStatus, error) {
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("Accounting data cannot be parsed %s ", stdout)
	}
	output := strings.Split(lines[0], "|")
	start := common.ParseDate(output[0])
	end := common.ParseDate(output[1])
	exitCode, err := strconv.Atoi(strings.Split(output[2], ":")[0])
	if err != nil {
		return nil, fmt.Errorf("Exitcode cannot be parsed %s ", output[0])
	}
	state := output[3]
	reason := state
	if output[4] != "" {
		reason = output[4]
	}
	return &JobStatus{ExitCode: exitCode, JobState: state, Reason: reason, EndTime: end, StarTime: start}, nil
}

func parseJobId(response string) (string, error) {
	value := "Submitted batch job "
	splitResponse := strings.Split(response, value)
	if len(splitResponse) > 1 {
		return strings.TrimSpace(splitResponse[1]), nil
	}
	return "", fmt.Errorf("Not submitted batch job id found: %s ", response)
}

func (s SlurmCmd) PullImageScript(command, imagePath, scriptPath string, env map[string]string) (uint64, error) {
	var size uint64
	filePath := file.GenerateTmpFile("/tmp", "pullimage", "sh")
	// open output file
	fo, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
	if err != nil {
		return size, err
	}
	var b bytes.Buffer
	var commands []string
	//make sure image dir exists
	makeDir := fmt.Sprintf("mkdir -p %s", path.Dir(imagePath))
	commands = append(commands, makeDir)

	if c, ok := env["CLUSTER_CONFIG"]; ok {
		fConfig, err := ioutil.ReadFile(c)
		if err != nil {
			return size, err
		}
		if _, err := fo.Write(fConfig); err != nil {
			return size, err
		}
	}
	commands = append(commands, command)
	commands = append(commands, "stat --print='FileSize:%s' "+imagePath)

	if err := writeLines(&b, commands); err != nil {
		return 0, err
	}

	if _, err := fo.Write(b.Bytes()); err != nil {
		return size, err
	}
	err = s.CopyTo(filePath, scriptPath)
	if err != nil {
		return size, fmt.Errorf("Error copying run file to Slurm cluster. %s ", err)
	}
	out, err := s.ExecCmd(scriptPath)
	if err != nil {
		return size, err
	}
	klog.Infof(out)
	sizeInt, err := strconv.Atoi(strings.Split(out, "FileSize:")[1])
	if err != nil {
		return size, nil
	}
	size = uint64(sizeInt)
	return size, nil
}
