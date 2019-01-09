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

package file

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func EnsurePathExist(path string) error {
	if !CheckFileExist(path) {
		err := CreatePathDirectory(path)
		if err != nil {
			return err
		}
	}
	return nil
}

func CheckFileExist(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}
	return true
}

func FileSize(path string) (int64, error) {
	file, err := os.Open(path)
	if err != nil {
		// handle the error here
		return 0, err
	}
	defer file.Close()

	// get the file size
	stat, err := file.Stat()
	if err != nil {
		return 0, err
	}

	return stat.Size(), nil
}

//Cretate directory/es from full path
func CreatePathDirectory(path string) error {
	return os.MkdirAll(path, 0755)
}

func GenerateTmpFile(path, name, extension string) string {
	return fmt.Sprintf("%s/%s_job_%s.%s", path, name, time.Now().Format("20060102150405"), extension)
}

//Generate pod log directory path
func GenerateLogDir(sandboxLogDir string, sandboxID string) string {
	result := ""
	if sandboxLogDir != "" {
		result = sandboxLogDir
	} else {
		result = result + "/var/log/pods/" + sandboxID
	}
	return result
}

var CopyLocalImage = copyLocalFile //Allows to mock it

//Copy image from one filesystem path to another.
// destiny path to locate the file
// source path origin
func copyLocalFile(destiny string, source string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(destiny)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

var DownloaVirgoContainer = downloadAndUncompressFile //Allows to mock it

// Download an file and uncompress it
// destiny path to locate the file
// source remote url
func downloadAndUncompressFile(destiny string, source string) (err error) {

	// Create the file
	out, err := os.Create(destiny)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(source)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// uncompress
	uncompress, err := gzip.NewReader(resp.Body)
	// Writer the body to file
	_, err = io.Copy(out, uncompress)
	if err != nil {
		return err
	}

	return nil
}
