// Copyright 2026 Cisco Systems, Inc. and its affiliates
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
//
// SPDX-License-Identifier: Apache-2.0

package modelscan

import "fmt"

const (
	Scans       = "scans"
	ScanObjects = "objects"
)

func ScanByID(scanID string) string {
	return fmt.Sprintf("%s/%s", Scans, scanID)
}

func ObjectByID(scanID, objectID string) string {
	return fmt.Sprintf("%s/%s/%s", ScanByID(scanID), ScanObjects, objectID)
}

func MultipartPartURLs(scanID, objectID string) string {
	return ObjectByID(scanID, objectID) + "/multipart/part-urls"
}

func MultipartComplete(scanID, objectID string) string {
	return ObjectByID(scanID, objectID) + "/multipart/complete"
}

func MultipartAbort(scanID, objectID string) string {
	return ObjectByID(scanID, objectID) + "/multipart/abort"
}
