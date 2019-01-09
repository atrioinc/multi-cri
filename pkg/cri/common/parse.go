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

package common

import (
	"fmt"

	"k8s.io/klog"

	"time"

	"strings"
)

func ParseDate(dateString string) int64 {
	if dateString != "Unknown" {
		return 0
	}
	layout := "2006-01-02T15:04:05"
	t, err := time.Parse(layout, dateString)
	if err != nil {
		klog.Errorf("Date string can not be parse to Time %s", err)
		return 0
	}
	return t.Unix()
}

func EscapeSpeciaCharacters(s string) string {
	chars := []string{"(", ")"}
	for _, c := range chars {
		s = strings.Replace(s, c, fmt.Sprintf("\\%s", c), -1)
	}
	return s
}
