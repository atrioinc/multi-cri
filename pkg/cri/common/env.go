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
	"os"
	"strconv"
)

func GetEnv(name string, defaultString *string) string {
	value := os.Getenv(name)
	if value == "" {
		if defaultString != nil {
			return *defaultString
		}
		panic(fmt.Sprintf("%s env variable is required", name))
	}
	return value
}

func GetBoolEnv(name string, defaultString *bool) bool {
	value := os.Getenv(name)
	if value == "" {
		if defaultString != nil {
			return *defaultString
		}
		panic(fmt.Sprintf("%s env variable is required", name))
	}
	b, err := strconv.ParseBool(value)
	if err != nil {
		panic(err)
	}
	return b
}
