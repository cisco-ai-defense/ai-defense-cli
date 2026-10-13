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
	"strconv"
	"time"
)

// FlexInt64 handles JSON values that may be either a number or a string.
type FlexInt64 int64

func (f *FlexInt64) UnmarshalJSON(data []byte) error {
	var n int64
	if err := json.Unmarshal(data, &n); err == nil {
		*f = FlexInt64(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*f = FlexInt64(n)
	return nil
}

// URLType represents the type of model URL provider.
type URLType int

const (
	URLTypeNone        URLType = 0
	URLTypeHuggingFace URLType = 1
)

// ScanStatus represents the current status of a scan.
type ScanStatus string

const (
	ScanStatusNone        ScanStatus = "NONE_SCAN_STATUS"
	ScanStatusPending     ScanStatus = "PENDING"
	ScanStatusInProgress  ScanStatus = "IN_PROGRESS"
	ScanStatusCompleted   ScanStatus = "COMPLETED"
	ScanStatusFailed      ScanStatus = "FAILED"
	ScanStatusCanceled    ScanStatus = "CANCELED"
	ScanStatusSkipped     ScanStatus = "SKIPPED"
	ScanStatusDownloading ScanStatus = "DOWNLOADING"
)

// AnalysisType represents the type of analysis performed.
type AnalysisType string

const (
	AnalysisTypeNone       AnalysisType = "NONE_ANALYSIS_TYPE"
	AnalysisTypeFile       AnalysisType = "FILE_ANALYSIS"
	AnalysisTypeRepository AnalysisType = "REPOSITORY_ANALYSIS"
)

// Severity represents the severity level of a detected threat.
type Severity string

const (
	SeverityNone     Severity = "NONE_SEVERITY"
	SeveritySafe     Severity = "SAFE"
	SeverityLow      Severity = "LOW"
	SeverityMedium   Severity = "MEDIUM"
	SeverityHigh     Severity = "HIGH"
	SeverityCritical Severity = "CRITICAL"
)

// RiskCategory represents risk category classification for files.
type RiskCategory string

const (
	RiskCategoryNone       RiskCategory = "NONE_RISK_CATEGORY"
	RiskCategoryVulnerable RiskCategory = "VULNERABLE"
	RiskCategoryNoThreats  RiskCategory = "NO_THREATS"
	RiskCategoryNotScanned RiskCategory = "NOT_SCANNED"
)

// FileSubcategory holds supported file subcategory information.
type FileSubcategory struct {
	Name           string   `json:"name"`
	FileExtensions []string `json:"file_extensions"`
}

func (f *FileSubcategory) UnmarshalJSON(data []byte) error {
	type alias FileSubcategory
	var aux struct {
		alias
		FileExtensionsCamel []string `json:"fileExtensions"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*f = FileSubcategory(aux.alias)
	if len(f.FileExtensions) == 0 && len(aux.FileExtensionsCamel) > 0 {
		f.FileExtensions = aux.FileExtensionsCamel
	}
	return nil
}

// FileCategory holds supported file category information.
type FileCategory struct {
	Name          string            `json:"name"`
	Subcategories []FileSubcategory `json:"subcategories"`
}

// SupportedFileTypes holds supported file types organized by categories.
type SupportedFileTypes struct {
	Categories []FileCategory `json:"categories"`
}

// RegisterScanResponse is returned when a new scan is registered.
type RegisterScanResponse struct {
	ScanID             string             `json:"scan_id"`
	SupportedFileTypes SupportedFileTypes `json:"supported_file_types"`
}

func (r *RegisterScanResponse) UnmarshalJSON(data []byte) error {
	type alias RegisterScanResponse
	var aux struct {
		alias
		ScanIDCamel             string             `json:"scanId"`
		SupportedFileTypesCamel SupportedFileTypes `json:"supportedFileTypes"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*r = RegisterScanResponse(aux.alias)
	if r.ScanID == "" {
		r.ScanID = aux.ScanIDCamel
	}
	if len(r.SupportedFileTypes.Categories) == 0 && len(aux.SupportedFileTypesCamel.Categories) > 0 {
		r.SupportedFileTypes = aux.SupportedFileTypesCamel
	}
	return nil
}

// HuggingFaceAuth holds authentication for HuggingFace.
type HuggingFaceAuth struct {
	AccessToken string `json:"access_token"`
}

// Auth is an authentication container for different providers.
type Auth struct {
	HuggingFace *HuggingFaceAuth `json:"huggingface,omitempty"`
}

// FileObject represents a file to be scanned.
type FileObject struct {
	FileName string `json:"file_name"`
}

// URLObject represents a URL to be scanned.
type URLObject struct {
	URL       string  `json:"url"`
	Type      URLType `json:"type"`
	VersionID string  `json:"version_id,omitempty"`
	Auth      *Auth   `json:"auth,omitempty"`
}

// ScanObject represents either a file or URL to be scanned.
type ScanObject struct {
	FileObject *FileObject `json:"file_object,omitempty"`
	URLObject  *URLObject  `json:"url_object,omitempty"`
}

// CreateScanObjectRequest is sent to create a scan object and get a presigned URL.
type CreateScanObjectRequest struct {
	FileName           string      `json:"file_name"`
	Size               int64       `json:"size,omitempty"`
	ScanObject         *ScanObject `json:"scan_object,omitempty"`
	UseMultipartUpload bool        `json:"use_multipart_upload,omitempty"`
}

// CreateScanObjectResponse is returned after creating a scan object.
type CreateScanObjectResponse struct {
	ObjectID        string           `json:"object_id"`
	UploadURL       string           `json:"upload_url,omitempty"`
	MultipartUpload *MultipartUpload `json:"multipart_upload,omitempty"`
}

func (r *CreateScanObjectResponse) UnmarshalJSON(data []byte) error {
	type alias CreateScanObjectResponse
	var aux struct {
		alias
		ObjectIDCamel        string           `json:"objectId"`
		UploadURLCamel       string           `json:"uploadUrl"`
		MultipartUploadCamel *MultipartUpload `json:"multipartUpload"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*r = CreateScanObjectResponse(aux.alias)
	if r.ObjectID == "" {
		r.ObjectID = aux.ObjectIDCamel
	}
	if r.UploadURL == "" {
		r.UploadURL = aux.UploadURLCamel
	}
	if r.MultipartUpload == nil {
		r.MultipartUpload = aux.MultipartUploadCamel
	}
	return nil
}

// MultipartUpload holds the server-created S3 multipart upload for a scan object.
type MultipartUpload struct {
	UploadID      string    `json:"upload_id"`
	PartSizeBytes FlexInt64 `json:"part_size_bytes"`
}

func (m *MultipartUpload) UnmarshalJSON(data []byte) error {
	type alias MultipartUpload
	var aux struct {
		alias
		UploadIDCamel      string    `json:"uploadId"`
		PartSizeBytesCamel FlexInt64 `json:"partSizeBytes"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*m = MultipartUpload(aux.alias)
	if m.UploadID == "" {
		m.UploadID = aux.UploadIDCamel
	}
	if m.PartSizeBytes == 0 {
		m.PartSizeBytes = aux.PartSizeBytesCamel
	}
	return nil
}

// GetMultipartUploadPartURLsRequest requests presigned URLs for a batch of parts.
type GetMultipartUploadPartURLsRequest struct {
	UploadID    string `json:"upload_id"`
	PartNumbers []int  `json:"part_numbers"`
}

// MultipartUploadPartURL is the presigned URL for one part.
type MultipartUploadPartURL struct {
	PartNumber int    `json:"part_number"`
	UploadURL  string `json:"upload_url"`
}

func (p *MultipartUploadPartURL) UnmarshalJSON(data []byte) error {
	type alias MultipartUploadPartURL
	var aux struct {
		alias
		PartNumberCamel int    `json:"partNumber"`
		UploadURLCamel  string `json:"uploadUrl"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*p = MultipartUploadPartURL(aux.alias)
	if p.PartNumber == 0 {
		p.PartNumber = aux.PartNumberCamel
	}
	if p.UploadURL == "" {
		p.UploadURL = aux.UploadURLCamel
	}
	return nil
}

// GetMultipartUploadPartURLsResponse lists presigned URLs for the requested parts.
type GetMultipartUploadPartURLsResponse struct {
	Parts []MultipartUploadPartURL `json:"parts"`
}

// CompletedMultipartUploadPart pairs an uploaded part number with its ETag.
type CompletedMultipartUploadPart struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
}

// CompleteMultipartUploadRequest finalizes a multipart upload.
type CompleteMultipartUploadRequest struct {
	UploadID string                         `json:"upload_id"`
	Parts    []CompletedMultipartUploadPart `json:"parts"`
}

// AbortMultipartUploadRequest releases an incomplete multipart upload.
type AbortMultipartUploadRequest struct {
	UploadID string `json:"upload_id"`
}

// ModelRepoConfig holds configuration for validating a repository URL.
type ModelRepoConfig struct {
	URL  string  `json:"url"`
	Type URLType `json:"type"`
	Auth *Auth   `json:"auth,omitempty"`
}

// ValidateModelURLResponse is returned after validating a repository URL.
type ValidateModelURLResponse struct {
	IsAccessible bool   `json:"is_accessible"`
	ErrorMessage string `json:"error_message,omitempty"`
}

func (r *ValidateModelURLResponse) UnmarshalJSON(data []byte) error {
	type alias ValidateModelURLResponse
	var aux struct {
		alias
		IsAccessibleCamel bool   `json:"isAccessible"`
		ErrorMessageCamel string `json:"errorMessage"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*r = ValidateModelURLResponse(aux.alias)
	if !r.IsAccessible {
		r.IsAccessible = aux.IsAccessibleCamel
	}
	if r.ErrorMessage == "" {
		r.ErrorMessage = aux.ErrorMessageCamel
	}
	return nil
}

// Paging holds pagination information.
type Paging struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func (p *Paging) UnmarshalJSON(data []byte) error {
	type alias Paging
	var aux struct {
		alias
		Count int `json:"count"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*p = Paging(aux.alias)
	if p.Limit == 0 && aux.Count > 0 {
		p.Limit = aux.Count
	}
	return nil
}

// MarshalJSON emits API wire paging: total, offset, and count (not limit).
func (p Paging) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Total  int `json:"total"`
		Offset int `json:"offset"`
		Count  int `json:"count"`
	}{
		Total:  p.Total,
		Offset: p.Offset,
		Count:  p.Limit,
	})
}

// ThreatInfo holds information about a detected threat.
type ThreatInfo struct {
	ID          string   `json:"id"`
	ThreatID    string   `json:"threat_id"`
	ThreatType  string   `json:"threat_type"`
	Severity    Severity `json:"severity"`
	Details     string   `json:"details"`
	Description string   `json:"description"`
}

func (t *ThreatInfo) UnmarshalJSON(data []byte) error {
	type alias ThreatInfo
	var aux struct {
		alias
		ThreatIDCamel   string `json:"threatId"`
		ThreatTypeCamel string `json:"threatType"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*t = ThreatInfo(aux.alias)
	if t.ThreatID == "" {
		t.ThreatID = aux.ThreatIDCamel
	}
	if t.ThreatType == "" {
		t.ThreatType = aux.ThreatTypeCamel
	}
	return nil
}

// SubTechnique holds sub-technique level grouping with threat evidence.
type SubTechnique struct {
	SubTechniqueID   string       `json:"sub_technique_id"`
	SubTechniqueName string       `json:"sub_technique_name"`
	Description      string       `json:"description"`
	Indicators       []string     `json:"indicators"`
	MaxSeverity      Severity     `json:"max_severity"`
	Items            []ThreatInfo `json:"items"`
}

func (s *SubTechnique) UnmarshalJSON(data []byte) error {
	type alias SubTechnique
	var aux struct {
		alias
		SubTechniqueIDCamel   string   `json:"subTechniqueId"`
		SubTechniqueNameCamel string   `json:"subTechniqueName"`
		MaxSeverityCamel      Severity `json:"maxSeverity"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*s = SubTechnique(aux.alias)
	if s.SubTechniqueID == "" {
		s.SubTechniqueID = aux.SubTechniqueIDCamel
	}
	if s.SubTechniqueName == "" {
		s.SubTechniqueName = aux.SubTechniqueNameCamel
	}
	if s.MaxSeverity == "" {
		s.MaxSeverity = aux.MaxSeverityCamel
	}
	return nil
}

// Technique holds technique-level grouping of threats.
type Technique struct {
	TechniqueID   string         `json:"technique_id"`
	TechniqueName string         `json:"technique_name"`
	Items         []SubTechnique `json:"items"`
}

func (t *Technique) UnmarshalJSON(data []byte) error {
	type alias Technique
	var aux struct {
		alias
		TechniqueIDCamel   string `json:"techniqueId"`
		TechniqueNameCamel string `json:"techniqueName"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*t = Technique(aux.alias)
	if t.TechniqueID == "" {
		t.TechniqueID = aux.TechniqueIDCamel
	}
	if t.TechniqueName == "" {
		t.TechniqueName = aux.TechniqueNameCamel
	}
	return nil
}

// ThreatInfoList holds hierarchical view of threats grouped by taxonomy.
type ThreatInfoList struct {
	Items  []Technique `json:"items"`
	Paging Paging      `json:"paging"`
}

// FileInfo holds metadata for an analyzed file and detected issues.
type FileInfo struct {
	Name    string         `json:"name"`
	Size    FlexInt64      `json:"size"`
	Status  ScanStatus     `json:"status"`
	Threats ThreatInfoList `json:"threats"`
	Reason  string         `json:"reason,omitempty"`
}

// AnalysisResult holds a list of analyzed files with pagination.
type AnalysisResult struct {
	Items  []FileInfo `json:"items"`
	Paging Paging     `json:"paging"`
}

// RepositoryInfo holds details about a scanned repository.
type RepositoryInfo struct {
	URL          string `json:"url"`
	Version      string `json:"version"`
	FilesScanned int    `json:"files_scanned"`
}

func (r *RepositoryInfo) UnmarshalJSON(data []byte) error {
	type alias RepositoryInfo
	var aux struct {
		alias
		FilesScannedCamel int `json:"filesScanned"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*r = RepositoryInfo(aux.alias)
	if r.FilesScanned == 0 {
		r.FilesScanned = aux.FilesScannedCamel
	}
	return nil
}

// ScanStatusInfo holds comprehensive status information for a scan.
type ScanStatusInfo struct {
	ScanID          string          `json:"scan_id"`
	Status          ScanStatus      `json:"status"`
	CreatedAt       time.Time       `json:"created_at"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
	Type            AnalysisType    `json:"type"`
	Repository      *RepositoryInfo `json:"repository,omitempty"`
	AnalysisResults AnalysisResult  `json:"analysis_results"`
}

func (s *ScanStatusInfo) UnmarshalJSON(data []byte) error {
	type alias ScanStatusInfo
	var aux struct {
		alias
		ScanIDCamel          string          `json:"scanId"`
		CreatedAtCamel       time.Time       `json:"createdAt"`
		CompletedAtCamel     *time.Time      `json:"completedAt"`
		AnalysisResultsCamel AnalysisResult  `json:"analysisResults"`
		RepositoryCamel      *RepositoryInfo `json:"repository"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*s = ScanStatusInfo(aux.alias)
	if s.ScanID == "" {
		s.ScanID = aux.ScanIDCamel
	}
	if s.CreatedAt.IsZero() && !aux.CreatedAtCamel.IsZero() {
		s.CreatedAt = aux.CreatedAtCamel
	}
	if s.CompletedAt == nil && aux.CompletedAtCamel != nil {
		s.CompletedAt = aux.CompletedAtCamel
	}
	if isAnalysisResultEmpty(s.AnalysisResults) && !isAnalysisResultEmpty(aux.AnalysisResultsCamel) {
		s.AnalysisResults = aux.AnalysisResultsCamel
	}
	if s.Repository == nil && aux.RepositoryCamel != nil {
		s.Repository = aux.RepositoryCamel
	}
	return nil
}

func isAnalysisResultEmpty(a AnalysisResult) bool {
	return len(a.Items) == 0 && a.Paging.Total == 0 && a.Paging.Limit == 0 && a.Paging.Offset == 0
}

// GetScanStatusRequest holds parameters for retrieving scan status.
type GetScanStatusRequest struct {
	FileLimit    int           `json:"file_limit,omitempty"`
	FileOffset   int           `json:"file_offset,omitempty"`
	Query        string        `json:"query,omitempty"`
	Severity     []Severity    `json:"severity,omitempty"`
	RiskCategory *RiskCategory `json:"risk_category,omitempty"`
}

// GetScanStatusResponse wraps the scan status info.
type GetScanStatusResponse struct {
	ScanStatusInfo ScanStatusInfo `json:"scan_status_info"`
}

func (r *GetScanStatusResponse) UnmarshalJSON(data []byte) error {
	type alias GetScanStatusResponse
	var aux struct {
		alias
		ScanStatusInfoCamel ScanStatusInfo `json:"scanStatusInfo"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*r = GetScanStatusResponse(aux.alias)
	if r.ScanStatusInfo.Status == "" && aux.ScanStatusInfoCamel.Status != "" {
		r.ScanStatusInfo = aux.ScanStatusInfoCamel
	}
	return nil
}

// ScanSummary holds high-level summary information for a scan.
type ScanSummary struct {
	ScanID           string         `json:"scan_id"`
	Name             string         `json:"name"`
	Type             AnalysisType   `json:"type"`
	FilesScanned     int            `json:"files_scanned"`
	CreatedAt        time.Time      `json:"created_at"`
	IssuesBySeverity map[string]int `json:"issues_by_severity"`
	Status           ScanStatus     `json:"status"`
}

func (s *ScanSummary) UnmarshalJSON(data []byte) error {
	type alias ScanSummary
	var aux struct {
		alias
		ScanIDCamel           string         `json:"scanId"`
		FilesScannedCamel     int            `json:"filesScanned"`
		CreatedAtCamel        time.Time      `json:"createdAt"`
		IssuesBySeverityCamel map[string]int `json:"issuesBySeverity"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*s = ScanSummary(aux.alias)
	if s.ScanID == "" {
		s.ScanID = aux.ScanIDCamel
	}
	if s.FilesScanned == 0 {
		s.FilesScanned = aux.FilesScannedCamel
	}
	if s.CreatedAt.IsZero() && !aux.CreatedAtCamel.IsZero() {
		s.CreatedAt = aux.CreatedAtCamel
	}
	if len(s.IssuesBySeverity) == 0 && len(aux.IssuesBySeverityCamel) > 0 {
		s.IssuesBySeverity = aux.IssuesBySeverityCamel
	}
	return nil
}

// ScansPage holds a list of scan summaries with pagination.
type ScansPage struct {
	Items  []ScanSummary `json:"items"`
	Paging Paging        `json:"paging"`
}

// ListScansRequest holds parameters for listing scans.
type ListScansRequest struct {
	Limit    int          `json:"limit,omitempty"`
	Offset   int          `json:"offset,omitempty"`
	Name     string       `json:"name,omitempty"`
	Type     AnalysisType `json:"type,omitempty"`
	Severity []Severity   `json:"severity,omitempty"`
	Status   []ScanStatus `json:"status,omitempty"`
}

// ListScansResponse wraps the list of scans.
type ListScansResponse struct {
	Scans ScansPage `json:"scans"`
}
