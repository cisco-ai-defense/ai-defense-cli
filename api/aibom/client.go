package aibom

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

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

// CreateAnalysis submits an AIBOM analysis report.
func (c *Client) CreateAnalysis(req *CreateAnalysisRequest) (*CreateAnalysisResponse, error) {
	resp, err := c.base.Post(Analysis(), req)
	if err != nil {
		return nil, err
	}
	var result CreateAnalysisResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse create analysis response: %w", err)
	}
	return &result, nil
}

// SubmitReportFile submits a raw report JSON map as an analysis.
func (c *Client) SubmitReportFile(report map[string]interface{}) (json.RawMessage, error) {
	req, err := createAnalysisRequestFromReport(report)
	if err != nil {
		return nil, err
	}
	return c.base.Post(Analysis(), req)
}

func createAnalysisRequestFromReport(report map[string]interface{}) (*CreateAnalysisRequest, error) {
	if report == nil {
		return nil, fmt.Errorf("report payload is required")
	}

	metadata := extractReportMetadata(report)
	runID := getStringField(metadata, "run_id")
	if runID == "" {
		runID = fmt.Sprintf("aid-cli-%d", time.Now().UTC().UnixNano())
	}
	analyzerVersion := getStringField(metadata, "analyzer_version")
	if analyzerVersion == "" {
		analyzerVersion = getStringField(metadata, "tool_version")
	}
	if analyzerVersion == "" {
		analyzerVersion = "cisco-aid-cli"
	}

	submittedAt, err := parseSubmittedAt(metadata)
	if err != nil {
		return nil, fmt.Errorf("invalid submitted_at in report metadata: %w", err)
	}
	if submittedAt == nil {
		now := time.Now().UTC()
		submittedAt = &now
	}

	// source_kind is a protobuf enum on the server: an empty string is rejected
	// ("invalid value for enum field source_kind"). The CLI analyzes local
	// --source paths, so default to SOURCE_KIND_LOCAL_PATH while still honoring
	// an explicit value carried in the report metadata.
	sourceKind := SourceKind(getStringField(metadata, "source_kind"))
	if sourceKind == "" {
		sourceKind = SourceKindLocalPath
	}

	return &CreateAnalysisRequest{
		RunID:           runID,
		AnalyzerVersion: analyzerVersion,
		SubmittedAt:     submittedAt,
		SourceKind:      sourceKind,
		Report:          report,
	}, nil
}

func parseSubmittedAt(metadata map[string]interface{}) (*time.Time, error) {
	raw, ok := metadata["submitted_at"]
	if !ok || raw == nil {
		return nil, nil
	}

	switch v := raw.(type) {
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return nil, nil
		}
		parsed, err := time.Parse(time.RFC3339, s)
		if err != nil {
			parsed, err = time.Parse(time.RFC3339Nano, s)
			if err != nil {
				return nil, err
			}
		}
		utc := parsed.UTC()
		return &utc, nil
	case time.Time:
		utc := v.UTC()
		return &utc, nil
	default:
		return nil, fmt.Errorf("unsupported type %T", raw)
	}
}

func extractReportMetadata(report map[string]interface{}) map[string]interface{} {
	root := report
	if nested := getMapField(report, "aibom_analysis"); nested != nil {
		root = nested
	} else if nested := getMapField(report, "aiswf_analysis"); nested != nil {
		root = nested
	}
	if metadata := getMapField(root, "metadata"); metadata != nil {
		return metadata
	}
	return map[string]interface{}{}
}

func getMapField(m map[string]interface{}, key string) map[string]interface{} {
	v, ok := m[key]
	if !ok {
		return nil
	}
	mv, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	return mv
}

func getStringField(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(s)
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
