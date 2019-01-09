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

package ssh

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"cri-babelfish/pkg/cri/common"

	"k8s.io/klog"
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

func TestIntegrationAuthFileKey(t *testing.T) {
	defer recoverEnv(t)
	keypath := common.GetEnv("TEST_SSH_KEY_PATH", nil)
	user := common.GetEnv("TEST_SSH_USER", nil)
	host := common.GetEnv("TEST_SSH_HOST", nil)
	port := common.GetEnv("TEST_SSH_PORT", nil)
	adapter := NewSSH(user, host, port, &keypath, nil, nil)
	season, err := adapter.GetSession(true)
	if err != nil {
		t.Errorf("Error on auth method through key file")
	} else {
		season.Close()
	}
}

func TestIntegrationAuthPassword(t *testing.T) {
	defer recoverEnv(t)
	password := common.GetEnv("TEST_SSH_PASSWORD", nil)
	user := common.GetEnv("TEST_SSH_USER", nil)
	host := common.GetEnv("TEST_SSH_HOST", nil)
	port := common.GetEnv("TEST_SSH_PORT", nil)
	adapter := NewSSH(user, host, port, nil, &password, nil)
	season, err := adapter.GetSession(true)
	if err != nil {
		t.Errorf("Error on auth method through key file")
	} else {
		season.Close()
	}
}

func TestIntegrationAuthKey(t *testing.T) {
	defer recoverEnv(t)
	key := common.GetEnv("TEST_SSH_KEY", nil)
	user := common.GetEnv("TEST_SSH_USER", nil)
	host := common.GetEnv("TEST_SSH_HOST", nil)
	port := common.GetEnv("TEST_SSH_PORT", nil)

	adapter := NewSSH(user, host, port, nil, nil, []byte(key))
	season, err := adapter.GetSession(true)
	if err != nil {
		t.Errorf("Error on auth method through key file")
	} else {
		season.Close()
	}
}

func TestIntegrationRun(t *testing.T) {
	defer recoverEnv(t)
	keypath := common.GetEnv("TEST_SSH_KEY_PATH", nil)
	user := common.GetEnv("TEST_SSH_USER", nil)
	host := common.GetEnv("TEST_SSH_HOST", nil)
	port := common.GetEnv("TEST_SSH_PORT", nil)
	adapter := NewSSH(user, host, port, &keypath, nil, nil)

	stdout, _, err := adapter.Run("echo \"Hello ssh\"", nil, nil, false)
	if err != nil {
		t.Errorf("Error running command: %s", err)
	}
	if stdout == "Hello ssh\n" {
		fmt.Printf("Test ok")
	} else {
		t.Errorf("Error on stdout: %s", stdout)
	}

}

func TestIntegrationCopy(t *testing.T) {
	defer recoverEnv(t)
	keypath := common.GetEnv("TEST_SSH_KEY_PATH", nil)
	user := common.GetEnv("TEST_SSH_USER", nil)
	host := common.GetEnv("TEST_SSH_HOST", nil)
	port := common.GetEnv("TEST_SSH_PORT", nil)
	adapter := NewSSH(user, host, port, &keypath, nil, nil)

	source := "testdata/testfile.txt"
	destination := "testfile-copy.txt"
	err := adapter.CopyTo(source, destination)
	if err != nil {
		t.Errorf("Error copying file to cluster: %s", err)
	}
	source = destination
	destination = "/tmp/file.txt"
	err = adapter.CopyFrom(source, destination)
	if err != nil {
		t.Errorf("Error copying file from cluster: %s", err)
	}

}

func TestIntegrationLog(t *testing.T) {
	defer recoverEnv(t)
	keypath := common.GetEnv("TEST_SSH_KEY_PATH", nil)
	user := common.GetEnv("TEST_SSH_USER", nil)
	host := common.GetEnv("TEST_SSH_HOST", nil)
	port := common.GetEnv("TEST_SSH_PORT", nil)
	adapter := NewSSH(user, host, port, &keypath, nil, nil)

	fout, err := os.OpenFile("TestLog-out.txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		t.Errorf("Error opening output log file: %s", err)
	}

	ferr, err := os.OpenFile("TestLog-err.txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		t.Errorf("Error opening output stderr file: %s", err)
	}
	stdoutString, _, err := adapter.Run("echo \"hello\" && sleep 1 && echo \"world\"", fout, ferr, false)
	if err != nil {
		t.Errorf("Error on Run: %s", err)
	}
	fout.Sync()
	fout.Seek(0, 0)
	stdoutbuffer := new(bytes.Buffer)
	bytesread, err := stdoutbuffer.ReadFrom(fout)
	if err != nil {
		t.Errorf("Error reading the logs file")
	}
	outstring := stdoutbuffer.String()
	if outstring != "hello\nworld\n" {
		t.Errorf("Error the output stdout does not match the expected (bytes read %d) : %s", bytesread, outstring)
	}
	if stdoutString != "hello\nworld\n" {

	}
	ferr.Close()
	fout.Close()
}

func TestIntegrationAsync(t *testing.T) {
	defer recoverEnv(t)
	keypath := common.GetEnv("TEST_SSH_KEY_PATH", nil)
	user := common.GetEnv("TEST_SSH_USER", nil)
	host := common.GetEnv("TEST_SSH_HOST", nil)
	port := common.GetEnv("TEST_SSH_PORT", nil)
	adapter := NewSSH(user, host, port, &keypath, nil, nil)

	fout, err := os.OpenFile("TestAsync-out.txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		t.Errorf("Error opening output log file: %s", err)
	} else {
		defer fout.Close()
	}

	ferr, err := os.OpenFile("TestAsync-err.txt", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		t.Errorf("Error opening output stderr file: %s", err)
	} else {
		defer fout.Close()
	}
	start := time.Now()
	session, err := adapter.RunAsync("echo \"hello\" && sleep 30 && echo \"world\"", fout, ferr)
	if err != nil {
		t.Errorf("Error on runasync: %s", err)
	} else {
		defer session.Close()
	}
	end := time.Now()
	eleapsed := end.Sub(start)
	if eleapsed.Seconds() >= 30 {
		t.Errorf("Command was not run async, eleapsed %f", eleapsed.Seconds())
	}

	stdoutbuffer := new(bytes.Buffer)
	bytesread := int64(0)
	for i := 0; i < 3 && bytesread < 6; i++ {
		time.Sleep(5 * time.Second)
		fout.Sync()
		fout.Seek(0, 0)
		bytesread, err = stdoutbuffer.ReadFrom(fout)
		klog.Infof("Reading from stdout... : %s", stdoutbuffer.String())
	}
	if err != nil {
		t.Errorf("Error reading the logs file")
	}
	if strings.Contains(stdoutbuffer.String(), "hello\n") {
		t.Errorf("Error the output stdout does not match the expected (bytes read %d) : %s", bytesread, stdoutbuffer.String())
	} else {
		klog.Infof("Stdout buffer ok! %s", stdoutbuffer.String())
	}
	session.Wait()

	fout.Sync()
	fout.Seek(0, 0)
	stdoutbuffer = new(bytes.Buffer)
	bytesread, err = stdoutbuffer.ReadFrom(fout)
	if err != nil {
		t.Errorf("Error reading the logs file")
	}
	if strings.Contains(stdoutbuffer.String(), "hello\nworld\n") {
		t.Errorf("Error the output stdout does not match the expected (bytes read %d) : %s", bytesread, stdoutbuffer.String())
	} else {
		klog.Infof("Stdout buffer ok! %s", stdoutbuffer.String())
	}
}
