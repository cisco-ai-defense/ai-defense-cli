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

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/cisco-ai-defense/ai-defense-cli/api"
)

const (
	defaultRetryCount = 60
	defaultWaitSecs   = 10

	envRetryCount = "AI_DEFENSE_RETRY_COUNT_FOR_SCANNING"
	envWaitSecs   = "AI_DEFENSE_WAIT_TIME_SECS_SUCCESSIVE_SCAN_INFO_CHECK"

	// scan-manager allows a scan up to 4 h after trigger; waiting slightly
	// longer returns its final status instead of a client-side timeout.
	// Matches DEFAULT_SCAN_RESULT_TIMEOUT_SECONDS in cisco-aidefense-sdk.
	scanResultTimeoutSecs = 15000

	// DefaultUploadTimeoutSecs bounds each presigned upload request (a whole
	// file or one multipart part) when the config does not set UploadTimeout.
	DefaultUploadTimeoutSecs = 300
)

// Client provides access to the AI Defense Model Scan API.
type Client struct {
	base   *api.BaseClient
	events ScanEvents
}

// NewClient creates a new model scan client.
func NewClient(apiKey string, config *api.Config) *Client {
	if config == nil {
		config = api.NewConfig("")
	}
	if config.UploadTimeout <= 0 {
		withUpload := *config
		withUpload.UploadTimeout = DefaultUploadTimeoutSecs
		config = &withUpload
	}
	return &Client{base: api.NewBaseClient(apiKey, config)}
}

// RegisterScan registers a new scan session.
func (c *Client) RegisterScan() (*RegisterScanResponse, error) {
	resp, err := c.base.Post(Scans+"/register", nil)
	if err != nil {
		return nil, err
	}
	var result RegisterScanResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse register scan response: %w", err)
	}
	return &result, nil
}

// CreateScanObject creates a scan object and returns (objectID, uploadURL).
func (c *Client) CreateScanObject(scanID string, req *CreateScanObjectRequest) (string, string, error) {
	result, err := c.createScanObject(scanID, req)
	if err != nil {
		return "", "", err
	}
	return result.ObjectID, result.UploadURL, nil
}

func (c *Client) createScanObject(scanID string, req *CreateScanObjectRequest) (*CreateScanObjectResponse, error) {
	resp, err := c.base.Post(ScanByID(scanID)+"/"+ScanObjects, req)
	if err != nil {
		return nil, err
	}
	var result CreateScanObjectResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse create scan object response: %w", err)
	}
	return &result, nil
}

// UploadFile uploads a file to the presigned URL.
func (c *Client) UploadFile(uploadURL string, filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	return c.base.DoRawPUT(uploadURL, f, info.Size())
}

// UploadScanResult uploads scan results for a specific scan object.
func (c *Client) UploadScanResult(scanID, objectID string, scanResult map[string]interface{}) error {
	body := map[string]interface{}{
		"scan_result": scanResult,
	}
	_, err := c.base.Post(ObjectByID(scanID, objectID)+"/results", body)
	return err
}

// MarkScanCompleted marks a scan as completed.
func (c *Client) MarkScanCompleted(scanID string, errors string) error {
	body := map[string]string{"errors": errors}
	_, err := c.base.Put(ScanByID(scanID)+"/complete", body)
	return err
}

// TriggerScan triggers execution of a scan for uploaded files.
func (c *Client) TriggerScan(scanID string) error {
	_, err := c.base.Put(ScanByID(scanID)+"/run", nil)
	return err
}

// ValidateScanURL validates a repository URL for scanning.
func (c *Client) ValidateScanURL(scanID string, req *ModelRepoConfig) (*ValidateModelURLResponse, error) {
	resp, err := c.base.Post(ScanByID(scanID)+"/validate_url", req)
	if err != nil {
		return nil, err
	}
	var result ValidateModelURLResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse validate URL response: %w", err)
	}
	return &result, nil
}

func scanStatusParams(req *GetScanStatusRequest) url.Values {
	params := url.Values{}
	if req == nil {
		return params
	}
	if req.FileLimit > 0 {
		params.Set("file_limit", strconv.Itoa(req.FileLimit))
	}
	if req.FileOffset > 0 {
		params.Set("file_offset", strconv.Itoa(req.FileOffset))
	}
	if req.Query != "" {
		params.Set("query", req.Query)
	}
	return params
}

func listScansParams(req *ListScansRequest) url.Values {
	params := url.Values{}
	if req == nil {
		return params
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.Itoa(req.Limit))
	}
	if req.Offset > 0 {
		params.Set("offset", strconv.Itoa(req.Offset))
	}
	if req.Name != "" {
		params.Set("name", req.Name)
	}
	if req.Type != "" {
		params.Set("type", string(req.Type))
	}
	return params
}

// GetScanRaw returns the verbatim scan status JSON from the API.
func (c *Client) GetScanRaw(scanID string, req *GetScanStatusRequest) (json.RawMessage, error) {
	return c.base.Get(ScanByID(scanID), scanStatusParams(req))
}

// GetScan retrieves detailed information about a specific scan.
func (c *Client) GetScan(scanID string, req *GetScanStatusRequest) (*GetScanStatusResponse, error) {
	resp, err := c.GetScanRaw(scanID, req)
	if err != nil {
		return nil, err
	}
	var result GetScanStatusResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse scan status response: %w", err)
	}
	return &result, nil
}

// ListScansRaw returns the verbatim list-scans JSON from the API.
func (c *Client) ListScansRaw(req *ListScansRequest) (json.RawMessage, error) {
	return c.base.Get(Scans, listScansParams(req))
}

// ListScans lists scans with pagination.
func (c *Client) ListScans(req *ListScansRequest) (*ListScansResponse, error) {
	resp, err := c.ListScansRaw(req)
	if err != nil {
		return nil, err
	}
	var result ListScansResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse list scans response: %w", err)
	}
	return &result, nil
}

// DeleteScan deletes a scan session.
func (c *Client) DeleteScan(scanID string) error {
	_, err := c.base.Delete(ScanByID(scanID))
	return err
}

// CancelScan cancels a running scan.
func (c *Client) CancelScan(scanID string) error {
	_, err := c.base.Post(ScanByID(scanID)+"/cancel", nil)
	return err
}

// CleanupScanData deletes scan data on failure.
func (c *Client) CleanupScanData(scanID string) {
	_ = c.DeleteScan(scanID)
}

// ScanFile performs a cloud-based file scan with default upload options.
func (c *Client) ScanFile(filePath string) (json.RawMessage, error) {
	return c.ScanFileWithOptions(filePath, UploadOptions{})
}

// ScanFileWithOptions performs a cloud-based file scan: register, upload,
// trigger, and poll.
func (c *Client) ScanFileWithOptions(filePath string, opts UploadOptions) (json.RawMessage, error) {
	if _, err := opts.concurrency(); err != nil {
		return nil, err
	}
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}
	if info.Size() == 0 {
		return nil, errEmptyFile
	}

	reg, err := c.RegisterScan()
	if err != nil {
		return nil, fmt.Errorf("failed to register scan: %w", err)
	}
	scanID := reg.ScanID

	if err := validateSupportedFileType(info.Name(), reg.SupportedFileTypes); err != nil {
		c.CleanupScanData(scanID)
		return nil, err
	}

	if err := c.UploadFileWithOptions(scanID, filePath, opts); err != nil {
		c.CleanupScanData(scanID)
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	if err := c.TriggerScan(scanID); err != nil {
		c.CleanupScanData(scanID)
		return nil, fmt.Errorf("failed to trigger scan: %w", err)
	}

	return c.pollScanCompletion(scanID)
}

// ScanRepo performs a cloud-based repository scan: register, validate, trigger, and poll.
func (c *Client) ScanRepo(repoConfig *ModelRepoConfig) (json.RawMessage, error) {
	reg, err := c.RegisterScan()
	if err != nil {
		return nil, fmt.Errorf("failed to register scan: %w", err)
	}
	scanID := reg.ScanID

	validation, err := c.ValidateScanURL(scanID, repoConfig)
	if err != nil {
		c.CleanupScanData(scanID)
		return nil, fmt.Errorf("failed to validate repo URL: %w", err)
	}
	if !validation.IsAccessible || validation.ErrorMessage != "" {
		c.CleanupScanData(scanID)
		return nil, &api.ValidationError{Message: validation.ErrorMessage}
	}

	if err := c.TriggerScan(scanID); err != nil {
		c.CleanupScanData(scanID)
		return nil, fmt.Errorf("failed to trigger scan: %w", err)
	}

	return c.pollScanCompletion(scanID)
}

// pollPlan returns the poll interval and count. Either env var replaces the
// 15,000 s default with retry count × wait time.
func pollPlan() (waitSecs, retryCount int) {
	if os.Getenv(envRetryCount) != "" || os.Getenv(envWaitSecs) != "" {
		return envIntOrDefault(envWaitSecs, defaultWaitSecs), envIntOrDefault(envRetryCount, defaultRetryCount)
	}
	return defaultWaitSecs, scanResultTimeoutSecs / defaultWaitSecs
}

func (c *Client) pollScanCompletion(scanID string) (json.RawMessage, error) {
	waitSecs, retryCount := pollPlan()

	terminalStatuses := map[ScanStatus]bool{
		ScanStatusCompleted: true,
		ScanStatusFailed:    true,
		ScanStatusCanceled:  true,
	}

	if c.events.PollingStarted != nil {
		c.events.PollingStarted()
	}

	for i := 0; i < retryCount; i++ {
		time.Sleep(time.Duration(waitSecs) * time.Second)

		resp, err := c.GetScan(scanID, &GetScanStatusRequest{FileLimit: 100})
		if err != nil {
			return nil, fmt.Errorf("failed to get scan status: %w", err)
		}

		if terminalStatuses[resp.ScanStatusInfo.Status] {
			data, err := json.Marshal(resp)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal scan response: %w", err)
			}
			return data, nil
		}
	}

	return nil, &api.PollTimeoutError{
		ScanID:        scanID,
		BudgetSecs:    retryCount * waitSecs,
		StatusCommand: "cisco-aid-cli model-scan info --scan-id " + scanID,
		BudgetEnvVars: []string{envRetryCount, envWaitSecs},
	}
}

func envIntOrDefault(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}
