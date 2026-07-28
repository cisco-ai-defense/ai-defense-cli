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

package aibom

import "time"

// BomStatus represents the status of a BOM analysis.
type BomStatus string

const (
	BomStatusUnspecified         BomStatus = "BOM_STATUS_UNSPECIFIED"
	BomStatusCompleted           BomStatus = "BOM_STATUS_COMPLETED"
	BomStatusCompletedWithErrors BomStatus = "BOM_STATUS_COMPLETED_WITH_ERRORS"
	BomStatusFailed              BomStatus = "BOM_STATUS_FAILED"
	BomStatusSkipped             BomStatus = "BOM_STATUS_SKIPPED"
)

// ComponentCategory represents the category of a component.
type ComponentCategory string

const (
	ComponentCategoryUnspecified ComponentCategory = "COMPONENT_CATEGORY_UNSPECIFIED"
	ComponentCategoryModel       ComponentCategory = "COMPONENT_CATEGORY_MODEL"
	ComponentCategoryAgent       ComponentCategory = "COMPONENT_CATEGORY_AGENT"
	ComponentCategoryData        ComponentCategory = "COMPONENT_CATEGORY_DATA"
	ComponentCategoryPrompt      ComponentCategory = "COMPONENT_CATEGORY_PROMPT"
	ComponentCategoryTool        ComponentCategory = "COMPONENT_CATEGORY_TOOL"
	ComponentCategoryChain       ComponentCategory = "COMPONENT_CATEGORY_CHAIN"
	ComponentCategoryEmbedding   ComponentCategory = "COMPONENT_CATEGORY_EMBEDDING"
	ComponentCategoryMemory      ComponentCategory = "COMPONENT_CATEGORY_MEMORY"
	ComponentCategoryOther       ComponentCategory = "COMPONENT_CATEGORY_OTHER"
)

// SourceKind represents the type of source being analyzed.
type SourceKind string

const (
	SourceKindUnspecified SourceKind = "SOURCE_KIND_UNSPECIFIED"
	SourceKindLocalPath   SourceKind = "SOURCE_KIND_LOCAL_PATH"
	SourceKindContainer   SourceKind = "SOURCE_KIND_CONTAINER"
	SourceKindOther       SourceKind = "SOURCE_KIND_OTHER"
)

// SortBy represents the field to sort BOM results by.
type SortBy string

const (
	SortByUnspecified      SortBy = "SORT_BY_UNSPECIFIED"
	SortBySubmittedAt      SortBy = "SORT_BY_SUBMITTED_AT"
	SortByLastGeneratedAt  SortBy = "SORT_BY_LAST_GENERATED_AT"
	SortByAssetsDiscovered SortBy = "SORT_BY_ASSETS_DISCOVERED"
)

// SortOrder represents sort direction.
type SortOrder string

const (
	SortOrderAscending  SortOrder = "asc"
	SortOrderDescending SortOrder = "desc"
)

// Paging holds pagination information.
type Paging struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// AssetTypeCounts holds counts of different asset types.
type AssetTypeCounts struct {
	Models     int `json:"models"`
	Embeddings int `json:"embeddings"`
	Prompts    int `json:"prompts"`
	Agents     int `json:"agents"`
	Tools      int `json:"tools"`
	Chains     int `json:"chains"`
}

// BomDetailSummary holds summary information for a BOM detail.
type BomDetailSummary struct {
	TotalAssets     int              `json:"total_assets"`
	LastGeneratedAt *time.Time       `json:"last_generated_at,omitempty"`
	AssetTypes      *AssetTypeCounts `json:"asset_types,omitempty"`
}

// BomDetail holds detailed BOM information.
type BomDetail struct {
	AnalysisID  string            `json:"analysis_id"`
	SourceName  string            `json:"source_name"`
	SourceKind  SourceKind        `json:"source_kind"`
	GeneratedAt *time.Time        `json:"generated_at,omitempty"`
	Summary     *BomDetailSummary `json:"summary,omitempty"`
	Status      BomStatus         `json:"status"`
}

// ComponentRow holds a component entry in a BOM.
type ComponentRow struct {
	Name            string            `json:"name"`
	Details         string            `json:"details"`
	Category        ComponentCategory `json:"category"`
	FilePath        string            `json:"file_path"`
	LineNumber      int               `json:"line_number"`
	Framework       string            `json:"framework"`
	LastGeneratedAt *time.Time        `json:"last_generated_at,omitempty"`
}

// BomSummaryItem holds summary info for a BOM in a list.
type BomSummaryItem struct {
	AnalysisID       string     `json:"analysis_id"`
	SourceName       string     `json:"source_name"`
	SourceKind       SourceKind `json:"source_kind"`
	AssetsDiscovered int        `json:"assets_discovered"`
	LastGeneratedAt  *time.Time `json:"last_generated_at,omitempty"`
	Status           BomStatus  `json:"status"`
}

// ListBomsResponse contains a paginated list of BOMs.
type ListBomsResponse struct {
	Items  []BomSummaryItem `json:"items"`
	Paging *Paging          `json:"paging,omitempty"`
}

// ListBomComponentsResponse contains a paginated list of BOM components.
type ListBomComponentsResponse struct {
	Items  []ComponentRow `json:"items"`
	Paging *Paging        `json:"paging,omitempty"`
}

// BomSummaryStats holds aggregate BOM statistics.
type BomSummaryStats struct {
	TotalBoms           int `json:"total_boms"`
	Completed           int `json:"completed"`
	CompletedWithErrors int `json:"completed_with_errors"`
	Failed              int `json:"failed"`
	TotalAssets         int `json:"total_assets"`
}

// GetBomSummaryResponse wraps BOM summary statistics.
type GetBomSummaryResponse struct {
	Summary *BomSummaryStats `json:"summary,omitempty"`
}

// ListBomsRequest holds query parameters for listing BOMs.
type ListBomsRequest struct {
	Search     string     `json:"search,omitempty"`
	Status     BomStatus  `json:"status,omitempty"`
	SourceKind SourceKind `json:"source_kind,omitempty"`
	FromTime   string     `json:"from,omitempty"`
	ToTime     string     `json:"to,omitempty"`
	SortBy     SortBy     `json:"sort_by,omitempty"`
	Order      SortOrder  `json:"order,omitempty"`
	Limit      int        `json:"limit,omitempty"`
	Offset     int        `json:"offset,omitempty"`
}

// ListBomComponentsRequest holds query parameters for listing BOM components.
type ListBomComponentsRequest struct {
	Search        string            `json:"search,omitempty"`
	ComponentType ComponentCategory `json:"type,omitempty"`
	Framework     string            `json:"framework,omitempty"`
	Limit         int               `json:"limit,omitempty"`
	Offset        int               `json:"offset,omitempty"`
}

// BomsSummaryRequest holds query parameters for BOM summary stats.
type BomsSummaryRequest struct {
	Search     string     `json:"search,omitempty"`
	Status     BomStatus  `json:"status,omitempty"`
	SourceKind SourceKind `json:"source_kind,omitempty"`
	FromTime   string     `json:"from,omitempty"`
	ToTime     string     `json:"to,omitempty"`
}
