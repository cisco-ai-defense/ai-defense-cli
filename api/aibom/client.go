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

import (
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/cisco-ai-defense/ai-defense-cli/api"
)

// Client provides access to the AI Defense AIBOM API.
type Client struct {
	base *api.BaseClient
}

// NewClient creates a new AIBOM client.
func NewClient(apiKey string, config *api.Config) *Client {
	return &Client{base: api.NewBaseClient(apiKey, config)}
}

// ListBoms lists BOMs with optional filters.
func (c *Client) ListBoms(req *ListBomsRequest) (json.RawMessage, error) {
	params := url.Values{}
	if req != nil {
		if req.Search != "" {
			params.Set("search", req.Search)
		}
		if req.Status != "" {
			params.Set("status", string(req.Status))
		}
		if req.SourceKind != "" {
			params.Set("source_kind", string(req.SourceKind))
		}
		if req.FromTime != "" {
			params.Set("from", req.FromTime)
		}
		if req.ToTime != "" {
			params.Set("to", req.ToTime)
		}
		if req.SortBy != "" {
			params.Set("sort_by", string(req.SortBy))
		}
		if req.Order != "" {
			params.Set("order", string(req.Order))
		}
		if req.Limit > 0 {
			params.Set("limit", strconv.Itoa(req.Limit))
		}
		if req.Offset > 0 {
			params.Set("offset", strconv.Itoa(req.Offset))
		}
	}
	return c.base.Get(Boms(), params)
}

// GetBom retrieves a specific BOM by analysis ID.
func (c *Client) GetBom(analysisID string) (json.RawMessage, error) {
	return c.base.Get(BomByID(analysisID), nil)
}

// DeleteBom deletes a BOM by analysis ID.
func (c *Client) DeleteBom(analysisID string) error {
	_, err := c.base.Delete(BomByID(analysisID))
	return err
}

// ListBomComponents lists components of a specific BOM.
func (c *Client) ListBomComponents(analysisID string, req *ListBomComponentsRequest) (json.RawMessage, error) {
	params := url.Values{}
	if req != nil {
		if req.Search != "" {
			params.Set("search", req.Search)
		}
		if req.ComponentType != "" {
			params.Set("type", string(req.ComponentType))
		}
		if req.Framework != "" {
			params.Set("framework", req.Framework)
		}
		if req.Limit > 0 {
			params.Set("limit", strconv.Itoa(req.Limit))
		}
		if req.Offset > 0 {
			params.Set("offset", strconv.Itoa(req.Offset))
		}
	}
	return c.base.Get(BomComponents(analysisID), params)
}

// GetBomSummary retrieves aggregate BOM statistics.
func (c *Client) GetBomSummary(req *BomsSummaryRequest) (json.RawMessage, error) {
	params := url.Values{}
	if req != nil {
		if req.Search != "" {
			params.Set("search", req.Search)
		}
		if req.Status != "" {
			params.Set("status", string(req.Status))
		}
		if req.SourceKind != "" {
			params.Set("source_kind", string(req.SourceKind))
		}
		if req.FromTime != "" {
			params.Set("from", req.FromTime)
		}
		if req.ToTime != "" {
			params.Set("to", req.ToTime)
		}
	}
	return c.base.Get(BomsSummary(), params)
}
