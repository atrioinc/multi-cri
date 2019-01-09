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
	"io"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tmc/scp"
	"golang.org/x/crypto/ssh"
	"k8s.io/klog"
)

type SSH struct {
	client   *ssh.Client
	user     string
	host     string
	port     string
	keypath  *string
	key      []byte
	password *string
}

func publicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		klog.Errorf("Error reading private key file: %s", err)
		return nil
	}
	klog.V(4).Infof("Public key file opened")
	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		klog.Errorf("Error parsing private key: %s", err)
		return nil
	}
	klog.V(4).Infof("Public key read")
	return ssh.PublicKeys(key)
}

func publicKey(key []byte) ssh.AuthMethod {
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		klog.Errorf("Error parsing private key: %s", err)
		return nil
	}
	klog.V(4).Infof("Public key read")
	return ssh.PublicKeys(signer)
}

/*
SSH constructior.
Mandatory parameters:
	user, host and port for the ssh connection
Optional parameters:
	keypath: the path of a file containing the rsa key (leave it blank if not used)
	password: a password for loggin (leave it blank if not used)
	key: A byte array containing the rsa private key (leave it nil if not used),
*/
func NewSSH(user, host, port string, keypath, password *string, key []byte) SSH {
	var adapter SSH
	adapter.user = user
	adapter.host = host
	adapter.port = port
	adapter.key = key
	adapter.keypath = keypath
	adapter.password = password
	return adapter
}

/*
This method creates the connection that will support the ssh session and store
this connection in the adapter.client field.
*/
func (adapter *SSH) Connect() error {
	type HostKeyCallback func(hostname string, remote net.Addr, key ssh.PublicKey) error
	var (
		auth         []ssh.AuthMethod
		addr         string
		clientConfig *ssh.ClientConfig
		client       *ssh.Client
		err          error
	)
	auth = make([]ssh.AuthMethod, 0)
	if adapter.key != nil {
		auth = append(auth, publicKey(adapter.key))
		klog.V(4).Infof("Auth method added: Private key")
	}
	if adapter.keypath != nil {
		auth = append(auth, publicKeyFile(*adapter.keypath))
		klog.V(4).Infof("Auth method added: Private key file")
	}
	if adapter.password != nil {
		auth = append(auth, ssh.Password(*adapter.password))
		klog.V(4).Infof("Auth method added: Password")
	}

	clientConfig = &ssh.ClientConfig{
		User:    adapter.user,
		Auth:    auth,
		Timeout: 30 * time.Second,
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}
	addr = fmt.Sprintf("%s:%s", adapter.host, adapter.port)
	klog.V(4).Infof("Connecting to %s ...", addr)
	client, err = ssh.Dial("tcp", addr, clientConfig)
	if err != nil {
		return fmt.Errorf("Error connecting to %s:%s  : %s", adapter.host, adapter.port, err)
	}
	klog.V(4).Infof("Connected successfully!!!")
	adapter.client = client
	return nil
}

/*
This function creates a new session and return it. The method will create a new
connection to support the new ssh session if it does not already exists.
*/
func (adapter *SSH) GetSession(tty bool) (*ssh.Session, error) {

	klog.V(4).Infof("Openning a new session...")
	if adapter.client == nil {
		err := adapter.Connect()
		if err != nil {
			return nil, fmt.Errorf("Unable to create a session, could not connect: %v", err)
		}
	}
	session, err := adapter.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("Error creating a new session for %s@%s:%s : %v", adapter.user, adapter.host, adapter.port, err)
	}
	if tty {
		klog.V(4).Infof("Requesting Pseudo terminal (Pty)")
		// Set up terminal modes
		modes := ssh.TerminalModes{
			ssh.ECHO:          0,     // disable echoing
			ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
			ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
		}

		// Request pseudo terminal
		if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
			session.Close()
			klog.Fatalf("request for pseudo terminal failed: %v", err)
		}
	}
	klog.V(4).Infof("New session opened!")

	return session, nil
}

/*
This function runs a command throgh ssh sincronously.  It accepts two io.WriteCloser
interfaces where the stdout and stderr of the command will be written. In addition,  stdout and stderr of the
command are returned as strings.
*/
func (adapter SSH) Run(command string, stdout, stderr io.WriteCloser, tty bool) (string, string, error) {
	session, err := adapter.GetSession(tty)
	if err != nil {
		return "", "", fmt.Errorf("Unable to get a session: %s", err)
	}
	defer func() {
		session.Close()
		klog.V(4).Infof("Session closed.")
	}()

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	klog.V(4).Infof("Running command: %s", command)

	err = session.Run(command)

	out := stdoutBuf.String()
	errString := stderrBuf.String()

	if stderr != nil {
		stderr.Write(stderrBuf.Bytes())
	}
	if stdout != nil {
		stdout.Write(stdoutBuf.Bytes())
	}

	if len(errString) > 0 {
		return out, errString, fmt.Errorf("Command %s error ", errString)
	}

	if err != nil {
		return stdoutBuf.String(), stderrBuf.String(), fmt.Errorf("Error running the command %s : %v", command, err)
	}

	klog.V(4).Infof("Command %s finished", command)
	return out, errString, nil
}

/*
This function runs a command through ssh asyncronously. The function accepts two
io.WriteCloser interfaces where the command will ouput the stdout and stderr.
The function also return the seassion that is running  the command. It is under
the responability of the user of this function to wait for the command to end and
close the session.
*/
func (adapter SSH) RunAsync(command string, stdout io.WriteCloser, stderr io.WriteCloser) (*ssh.Session, error) {
	session, err := adapter.GetSession(true)
	if err != nil {
		klog.Errorf("Unable to get a session: %s", err)
		return nil, fmt.Errorf("Unable to get a session: %s", err)
	}
	session.Stdout = stdout
	session.Stderr = stderr
	klog.V(4).Infof("Launching command: %s", command)
	err = session.Start(command)
	if err != nil {
		klog.Errorf("Unable to launch the command %s : %s", command, err)
		return nil, fmt.Errorf("Unable to launch the command %s : %s", command, err)
	}
	klog.V(4).Infof("Command: %s launched!", command)
	return session, nil
}

/*
This function copies a file from the local filesystem to the remote host through ssh.
The file permissions are preserved and it overwrites the destination if already exists.
*/
func (adapter SSH) CopyTo(source, destination string) error {
	session, err := adapter.GetSession(false)
	if err != nil {
		klog.Errorf("Unable to get a session: %s", err)
		return fmt.Errorf("Unable to get a session: %v", err)
	}
	klog.V(4).Infof("Copying file from %s to %s@%s:%s...", source, adapter.user, adapter.host, destination)
	defer func() {
		session.Close()
		klog.V(4).Infof("Session closed.")
	}()
	err = scp.CopyPath(source, destination, session)
	if err != nil {
		return fmt.Errorf("Unable to copy file %s to %s:%s:%s: %v", source, adapter.host, adapter.port, destination, err)
	}
	klog.V(4).Infof("File %s copied successfully!!", source)
	return nil
}

/*
This function copies a file from the remote host to the local filesystem through ssh.
The file permissions are preserved and it overwrites the destination if already exists.
*/
func (adapter SSH) CopyFrom(source, destination string) error {

	klog.V(4).Infof("Getting file permissions of: %s@%s:%s", adapter.user, adapter.host, source)
	cmd := fmt.Sprintf("stat -c \"%%a\" %s", source)
	stdout, _, err := adapter.Run(cmd, nil, nil, false)
	if err != nil {
		klog.Errorf("Unable to get file permissions for file %s : %s", destination, err)
		return fmt.Errorf("Unable to get file permissions for file %s : %s", destination, err)
	}

	modeuint, err := strconv.ParseUint(strings.TrimSuffix(stdout, "\n"), 8, 32)
	if err != nil {
		klog.Errorf("Unable to parse file permissions %s: %s", stdout, err)
		return fmt.Errorf("Unable to get file permissions for file %s : %s", stdout, err)
	}
	mode := os.FileMode(modeuint)
	klog.V(4).Infof("Got %o file permissions", mode)
	session, err := adapter.GetSession(false)
	if err != nil {
		klog.Errorf("Unable to get a session: %s", err)
		return fmt.Errorf("Unable to get a session: %s", err)
	}
	defer func() {
		session.Close()
		klog.V(4).Infof("Session closed.")
	}()
	remotefile, err := session.StdoutPipe()
	if err != nil {
		klog.Errorf("Unable to get a pipe from %s:%s: %s", adapter.host, adapter.port, err)
		return fmt.Errorf("Unable to get a pipe from %s:%s: %s", adapter.host, adapter.port, err)
	}
	klog.V(4).Infof("Opened pipe with %s", adapter.host)
	localfile, err := os.OpenFile(destination, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, mode)
	if err != nil {
		klog.Errorf("Unable to create the file %s: %s", destination, err)
		return fmt.Errorf("Unable to create the file %s: %s", destination, err)
	}
	defer func() {
		localfile.Close()
		klog.V(4).Infof("File closed.")
	}()

	klog.V(4).Infof("Copying file from %s@%s:%s to %s ...", adapter.user, adapter.host, source, destination)
	cmd = fmt.Sprintf("cat %s", source)
	if err := session.Start(cmd); err != nil {
		klog.Errorf("Unable to launch the command %s : %s", cmd, err)
		return fmt.Errorf("Unable to launch the command %s : %s", cmd, err)
	}
	_, err = io.Copy(localfile, remotefile)
	if err != nil {
		klog.Errorf("Unable to copy file %s : %s", source, err)
		return fmt.Errorf("Unable to copy file %s : %s", source, err)
	}
	if err := session.Wait(); err != nil {
		klog.Errorf("Error waiting for the file to be copied : %s", err)
		return fmt.Errorf("Error waiting for the file to be copied : %s", err)
	}
	klog.V(4).Infof("File %s@%s:%s copied succesfully!!", adapter.user, adapter.host, source)
	return nil
}
