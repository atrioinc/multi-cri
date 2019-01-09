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

package network

import (
	"fmt"
	"os"
	"sync"

	cnins "github.com/containernetworking/plugins/pkg/ns"
	"golang.org/x/sys/unix"
)

//Network manager
type NetworkManagerInterface interface {
	OpenNetNamespace(path string) (NetworkNamespaceInterface, error)
}

//Network Namespace manager
type NetworkNamespaceInterface interface {
	CreateNetNS(path string) error
	Remove() error
	GetPath() string
}

type CNINetwork struct {
	NetNS
}

func (n *CNINetwork) OpenNetNamespace(path string) (NetworkNamespaceInterface, error) {
	var err error
	netNS := new(NetNS)
	netNS.CreateNetNS(path)
	return netNS, err
}

//Create CNINetwork object
func NewNetworkCNIManager() NetworkManagerInterface {
	n := new(CNINetwork)
	return n
}

// NetNS holds network namespace for sandbox
type NetNS struct {
	sync.Mutex
	ns     cnins.NetNS
	closed bool
}

//Create namespace if not exists
func (n *NetNS) CreateNetNS(path string) error {
	var err error

	if path == "" {
		err = n.newNetNS()
	} else {
		err = n.openNetNS(path)
		if err != nil {
			err = n.newNetNS()
		}
	}
	return err
}

// OpenNetNS opens a network namespace for the sandbox
func (n *NetNS) openNetNS(path string) error {
	netns, err := cnins.GetNS(path)
	if err != nil {
		return fmt.Errorf("failed to setup network namespace %v", err)
	}
	n.ns = netns
	return nil
}

// NewNetNS creates a network namespace for the sandbox
func (n *NetNS) newNetNS() error {
	netns, err := cnins.NewNS()
	if err != nil {
		return fmt.Errorf("failed to setup network namespace %v", err)
	}
	n.ns = netns
	return nil
}

// Remove removes network namepace if it exists and not closed. Remove is idempotent,
// meaning it might be invoked multiple times and provides consistent result.
func (n *NetNS) Remove() error {
	n.Lock()
	defer n.Unlock()
	if !n.closed {
		err := n.ns.Close()
		if err != nil {
			return err
		}
		err = n.ensureNSDelete()
		if err != nil {
			return err
		}
		n.closed = true
	}
	return nil
}

// GetPath returns network namespace path for sandbox container
func (n *NetNS) GetPath() string {
	n.Lock()
	defer n.Unlock()
	return n.ns.Path()
}

// Ensure network namespace is deleted
func (n *NetNS) ensureNSDelete() error {

	path := n.ns.Path()
	if _, err := os.Stat(path); err == nil {
		if err := unix.Unmount(path, unix.MNT_DETACH); err != nil {
			return fmt.Errorf("Failed to unmount namespace %s: %v", path, err)
		}
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("Failed to clean up namespace %s: %v", path, err)
		}
	}
	return nil
}
