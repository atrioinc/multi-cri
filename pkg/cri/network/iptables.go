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
	osexec "os/exec"
	"strings"

	"k8s.io/klog"
)

const (
	iptableExec = "iptables"
)

var EnsureIPTableRules = EnsureCNIIPTableRules //Allows to mock it

//Create iptable rules which are required for CNI pod comunication
func EnsureCNIIPTableRules(netInterface string) error {
	target1 := "CNI"
	err := addIPTableTarget(target1)
	if err != nil {
		return fmt.Errorf("Error when creating target %s. %v", target1, err)
	}
	rule1 := []string{"FORWARD", "-o", netInterface, "-m", "conntrack", "--ctstate", "RELATED,ESTABLISHED", "-j", "ACCEPT"}
	rule2 := []string{"FORWARD", "-o", netInterface, "-j", "CNI"}
	rule3 := []string{"FORWARD", "-i", netInterface, "!", "-o", "cni0", "-j", "ACCEPT"}
	rule4 := []string{"FORWARD", "-i", netInterface, "-o", "cni0", "-j", "ACCEPT"}
	err = ensureIPTableRuleExists(rule1)
	if err != nil {
		return fmt.Errorf("Error when creating rule %v. %v", rule1, err)
	}

	err = ensureIPTableRuleExists(rule2)
	if err != nil {
		return fmt.Errorf("Error when creating rule %v. %v", rule2, err)
	}
	err = ensureIPTableRuleExists(rule3)
	if err != nil {
		return fmt.Errorf("Error when creating rule %v. %v", rule3, err)
	}
	err = ensureIPTableRuleExists(rule4)
	if err != nil {
		return fmt.Errorf("Error when creating rule %v. %v", rule4, err)
	}
	return nil
}

//Create rule if it does not exit
func ensureIPTableRuleExists(rule []string) error {
	itExists := checkIPTableRule(rule)
	if !itExists {
		return createIPTableRule(rule)
	}
	return nil
}

//Create iptable target
func addIPTableTarget(targetDef string) error {
	commandCreate := generateIPTablesCommand("-N", []string{targetDef})
	klog.V(4).Infof("Creating iptable target %v", commandCreate)
	cmd := osexec.Command(commandCreate[0], commandCreate[1:]...)
	out, err := cmd.CombinedOutput() //execute without log file, to the standard system output
	if err != nil {
		if strings.Contains(string(out), "Chain already exists") {
			klog.Infof("Iptable target %s already exists", targetDef)
		} else {
			return fmt.Errorf("Error when creating iptable target %v", targetDef)
		}
	}
	return nil
}

//Create iptable rule
func createIPTableRule(rule []string) error {
	commandCreate := generateIPTablesCommand("-A", rule)
	klog.V(4).Infof("Creating iptable rule %v", commandCreate)
	cmd := osexec.Command(commandCreate[0], commandCreate[1:]...)
	cmd.Stdout = os.Stdout
	err := cmd.Run() //execute without log file, to the standard system output
	if err != nil {
		return fmt.Errorf("Error when creating iptable rule %v", commandCreate)
	}
	return nil
}

//Check if iptable rule exits
func checkIPTableRule(rule []string) bool {
	commandCheck := generateIPTablesCommand("-C", rule)
	klog.V(4).Infof("Check iptable rule %v", commandCheck)
	cmd := osexec.Command(commandCheck[0], commandCheck[1:]...)
	cmd.Stdout = os.Stdout
	err := cmd.Run() //execute without log file, to the standard system output
	if err == nil {
		return true
	} else {
		return false
	}
}

//Delete iptable rule
func deleteIPTableRule(rule []string) error {
	commandCreate := generateIPTablesCommand("-D", rule)
	klog.V(4).Infof("Deleting iptable rule %v", commandCreate)
	cmd := osexec.Command(commandCreate[0], commandCreate[1:]...)
	cmd.Stdout = os.Stdout
	err := cmd.Run() //execute without log file, to the standard system output
	if err != nil {
		return fmt.Errorf("Error when deleting iptable rule %v", commandCreate)
	}
	return nil
}

//Genare the iptable command
func generateIPTablesCommand(action string, args []string) []string {
	cmd := []string{iptableExec, action}
	for _, value := range args {
		cmd = append(cmd, value)
	}
	return cmd
}
