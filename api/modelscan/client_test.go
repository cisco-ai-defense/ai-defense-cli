package modelscan

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/cisco-ai-defense/ai-defense-cli/api"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient("test-key", api.NewConfig(srv.URL))
}

func TestListScansRaw_PreservesAPIBody(t *testing.T) {
	const body = `{"scans":{"items":[],"paging":{"total":1,"count":5,"offset":0}}}`
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	})

	raw, err := c.ListScansRaw(&ListScansRequest{Limit: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(raw) != body {
		t.Fatalf("expected verbatim API body, got %s", raw)
	}
}

func TestGetScanRaw_PreservesAPIBody(t *testing.T) {
	const body = `{"scan_status_info":{"scan_id":"s-1","status":"COMPLETED","analysis_results":{"items":[],"paging":{"total":0,"count":10,"offset":0}}}}`
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	})

	raw, err := c.GetScanRaw("s-1", &GetScanStatusRequest{FileLimit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(raw) != body {
		t.Fatalf("expected verbatim API body, got %s", raw)
	}
}

func TestPaging_MarshalJSONUsesCount(t *testing.T) {
	data, err := json.Marshal(Paging{Total: 34, Limit: 5, Offset: 0})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]int
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := got["limit"]; ok {
		t.Fatalf("expected no limit key, got %v", got)
	}
	if got["count"] != 5 {
		t.Fatalf("expected count=5, got %v", got)
	}
	if got["total"] != 34 {
		t.Fatalf("expected total=34, got %v", got)
	}
}

func TestListScans(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Query().Get("limit") != "5" {
			t.Errorf("expected limit=5, got %q", r.URL.Query().Get("limit"))
		}
		w.Write([]byte(`{"items":[],"paging":{"total":0}}`))
	})

	resp, err := c.ListScans(&ListScansRequest{Limit: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestGetScan(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Write([]byte(`{"scan_status_info":{"scan_id":"s-1","status":"COMPLETED"}}`))
	})

	resp, err := c.GetScan("s-1", &GetScanStatusRequest{FileLimit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ScanStatusInfo.ScanID != "s-1" {
		t.Errorf("expected scan_id 's-1', got %q", resp.ScanStatusInfo.ScanID)
	}
}

func TestGetScan_CamelCaseResponse(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Write([]byte(`{"scanStatusInfo":{"scanId":"s-2","status":"COMPLETED","analysisResults":{"items":[],"paging":{"total":0,"limit":0,"offset":0}}}}`))
	})

	resp, err := c.GetScan("s-2", &GetScanStatusRequest{FileLimit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ScanStatusInfo.ScanID != "s-2" {
		t.Errorf("expected scanId 's-2', got %q", resp.ScanStatusInfo.ScanID)
	}
	if resp.ScanStatusInfo.Status != ScanStatusCompleted {
		t.Errorf("expected COMPLETED status, got %q", resp.ScanStatusInfo.Status)
	}
}

func TestGetScan_CamelCaseResponse_EmptyItemsWithPaging(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Write([]byte(`{"scanStatusInfo":{"scanId":"s-4","status":"COMPLETED","analysisResults":{"items":[],"paging":{"total":42,"count":10,"offset":30}}}}`))
	})

	resp, err := c.GetScan("s-4", &GetScanStatusRequest{FileLimit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ScanStatusInfo.AnalysisResults.Paging.Total != 42 {
		t.Fatalf("expected paging total 42, got %d", resp.ScanStatusInfo.AnalysisResults.Paging.Total)
	}
	if resp.ScanStatusInfo.AnalysisResults.Paging.Limit != 10 {
		t.Fatalf("expected paging limit from count=10, got %d", resp.ScanStatusInfo.AnalysisResults.Paging.Limit)
	}
	if resp.ScanStatusInfo.AnalysisResults.Paging.Offset != 30 {
		t.Fatalf("expected paging offset 30, got %d", resp.ScanStatusInfo.AnalysisResults.Paging.Offset)
	}
}

func TestGetScan_CamelCaseThreatFieldsAndPagingCount(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Write([]byte(`{"scanStatusInfo":{"scanId":"s-3","status":"COMPLETED","analysisResults":{"items":[{"name":"file.bin","size":"10","status":"COMPLETED","threats":{"items":[{"techniqueId":"TA0001","techniqueName":"Initial Access","items":[{"subTechniqueId":"T1190","subTechniqueName":"Exploit Public-Facing Application","maxSeverity":"HIGH","items":[{"id":"th-1","threatId":"TH-100","threatType":"MALWARE","severity":"HIGH"}]}]}],"paging":{"total":1,"count":25,"offset":0}}}],"paging":{"total":1,"count":25,"offset":0}}}}`))
	})

	resp, err := c.GetScan("s-3", &GetScanStatusRequest{FileLimit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	file := resp.ScanStatusInfo.AnalysisResults.Items[0]
	if file.Threats.Paging.Limit != 25 {
		t.Fatalf("expected threats paging limit to decode from count=25, got %d", file.Threats.Paging.Limit)
	}
	technique := file.Threats.Items[0]
	if technique.TechniqueID != "TA0001" {
		t.Fatalf("expected techniqueId TA0001, got %q", technique.TechniqueID)
	}
	subTechnique := technique.Items[0]
	if subTechnique.SubTechniqueID != "T1190" {
		t.Fatalf("expected subTechniqueId T1190, got %q", subTechnique.SubTechniqueID)
	}
	threat := subTechnique.Items[0]
	if threat.ThreatID != "TH-100" {
		t.Fatalf("expected threatId TH-100, got %q", threat.ThreatID)
	}
	if threat.ThreatType != "MALWARE" {
		t.Fatalf("expected threatType MALWARE, got %q", threat.ThreatType)
	}
}

func TestRegisterScan(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Write([]byte(`{"scan_id":"new-scan"}`))
	})

	resp, err := c.RegisterScan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ScanID != "new-scan" {
		t.Errorf("expected 'new-scan', got %q", resp.ScanID)
	}
}

func TestRegisterScan_CamelCaseResponse(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Write([]byte(`{"scanId":"new-scan","supportedFileTypes":{"categories":[{"name":"model","subcategories":[{"name":"weights","fileExtensions":[".bin",".safetensors"]}]}]}}`))
	})

	resp, err := c.RegisterScan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ScanID != "new-scan" {
		t.Errorf("expected 'new-scan', got %q", resp.ScanID)
	}
	if got := len(resp.SupportedFileTypes.Categories); got != 1 {
		t.Fatalf("expected 1 category, got %d", got)
	}
	if got := len(resp.SupportedFileTypes.Categories[0].Subcategories[0].FileExtensions); got != 2 {
		t.Fatalf("expected 2 file extensions, got %d", got)
	}
}

func TestCreateScanObject_CamelCaseResponse(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Write([]byte(`{"objectId":"obj-1","uploadUrl":"https://example.com/presigned"}`))
	})

	objectID, uploadURL, err := c.CreateScanObject("s-1", &CreateScanObjectRequest{FileName: "test.bin"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if objectID != "obj-1" {
		t.Errorf("expected objectID obj-1, got %q", objectID)
	}
	if uploadURL != "https://example.com/presigned" {
		t.Errorf("expected uploadURL to decode from uploadUrl, got %q", uploadURL)
	}
}

func TestScanRepo_StopsOnValidationErrorMessage(t *testing.T) {
	triggerCalled := false
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path[len(r.URL.Path)-len("/register"):] == "/register":
			w.Write([]byte(`{"scan_id":"scan-1"}`))
		case r.Method == http.MethodPost && r.URL.Path[len(r.URL.Path)-len("/validate_url"):] == "/validate_url":
			w.Write([]byte(`{"is_accessible":true,"error_message":"no scannable files found"}`))
		case r.Method == http.MethodPut && r.URL.Path[len(r.URL.Path)-len("/run"):] == "/run":
			triggerCalled = true
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	_, err := c.ScanRepo(&ModelRepoConfig{URL: "https://huggingface.co/example/repo"})
	if err == nil {
		t.Fatal("expected validation error")
	}
	var valErr *api.ValidationError
	if !errors.As(err, &valErr) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if valErr.Message != "no scannable files found" {
		t.Errorf("expected validation message, got %q", valErr.Message)
	}
	if triggerCalled {
		t.Fatal("expected TriggerScan not to be called when validation returns error_message")
	}
}

func TestValidateScanURL_CamelCaseResponse(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Write([]byte(`{"isAccessible":false,"errorMessage":"repo is not reachable"}`))
	})

	resp, err := c.ValidateScanURL("s-1", &ModelRepoConfig{URL: "https://example.com/repo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.IsAccessible {
		t.Fatal("expected IsAccessible to be false")
	}
	if resp.ErrorMessage != "repo is not reachable" {
		t.Errorf("expected error message from errorMessage, got %q", resp.ErrorMessage)
	}
}

func TestListScans_CamelCaseResponse(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Write([]byte(`{"scans":{"items":[{"scanId":"scan-1","name":"scan-one","type":"FILE_ANALYSIS","filesScanned":2,"createdAt":"2026-06-05T00:00:00Z","issuesBySeverity":{"HIGH":1},"status":"COMPLETED"}],"paging":{"total":1,"count":10,"offset":0}}}`))
	})

	resp, err := c.ListScans(&ListScansRequest{Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(resp.Scans.Items); got != 1 {
		t.Fatalf("expected 1 item, got %d", got)
	}
	if resp.Scans.Items[0].ScanID != "scan-1" {
		t.Errorf("expected scanId to decode, got %q", resp.Scans.Items[0].ScanID)
	}
	if resp.Scans.Items[0].FilesScanned != 2 {
		t.Errorf("expected filesScanned to decode, got %d", resp.Scans.Items[0].FilesScanned)
	}
	if resp.Scans.Paging.Limit != 10 {
		t.Errorf("expected paging limit to decode from count=10, got %d", resp.Scans.Paging.Limit)
	}
}

func TestDeleteScan(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	err := c.DeleteScan("s-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFlexInt64_Number(t *testing.T) {
	var f FlexInt64
	if err := json.Unmarshal([]byte(`12345`), &f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if int64(f) != 12345 {
		t.Errorf("expected 12345, got %d", f)
	}
}

func TestFlexInt64_String(t *testing.T) {
	var f FlexInt64
	if err := json.Unmarshal([]byte(`"67890"`), &f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if int64(f) != 67890 {
		t.Errorf("expected 67890, got %d", f)
	}
}

func TestFlexInt64_InvalidString(t *testing.T) {
	var f FlexInt64
	if err := json.Unmarshal([]byte(`"not-a-number"`), &f); err == nil {
		t.Fatal("expected error for non-numeric string")
	}
}

func TestFlexInt64_InStruct(t *testing.T) {
	data := `{"name":"test.bin","size":"999","status":"COMPLETED","threats":{"items":[],"paging":{"total":0,"limit":0,"offset":0}}}`
	var fi FileInfo
	if err := json.Unmarshal([]byte(data), &fi); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if int64(fi.Size) != 999 {
		t.Errorf("expected size 999, got %d", fi.Size)
	}
}

func TestValidateSupportedFileType(t *testing.T) {
	types := SupportedFileTypes{
		Categories: []FileCategory{{
			Subcategories: []FileSubcategory{{
				FileExtensions: []string{".bin", ".safetensors"},
			}},
		}},
	}
	if err := validateSupportedFileType("model.bin", types); err != nil {
		t.Fatalf("expected .bin to be supported, got %v", err)
	}
	if err := validateSupportedFileType("archive.tgz", types); err == nil {
		t.Fatal("expected .tgz to be rejected")
	}
}

func TestHasSupportedExtension_CompoundSuffix(t *testing.T) {
	exts := map[string]struct{}{".tar": {}, ".gz": {}}
	if !hasSupportedExtension("weights.tar.gz", exts) {
		t.Fatal("expected .tar suffix match for compound extension")
	}
}

func TestTriggerScan(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	})
	if err := c.TriggerScan("scan-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCancelScan(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	})
	if err := c.CancelScan("scan-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMarkScanCompleted(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	})
	if err := c.MarkScanCompleted("scan-1", "none"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUploadScanResult(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	})
	err := c.UploadScanResult("scan-1", "obj-1", map[string]interface{}{"status": "ok"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScanFile_UnsupportedExtension(t *testing.T) {
	filePath := t.TempDir() + "/archive.zip"
	if err := os.WriteFile(filePath, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	deleteCalled := false
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/register") {
			w.Write([]byte(`{"scan_id":"scan-1","supported_file_types":{"categories":[{"subcategories":[{"file_extensions":[".bin"]}]}]}}`))
			return
		}
		if r.Method == http.MethodDelete {
			deleteCalled = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
	})

	_, err := c.ScanFile(filePath)
	if err == nil {
		t.Fatal("expected unsupported file type error")
	}
	if !deleteCalled {
		t.Fatal("expected cleanup delete after unsupported extension")
	}
}

func TestPollScanCompletion_Timeout(t *testing.T) {
	t.Setenv("AI_DEFENSE_WAIT_TIME_SECS_SUCCESSIVE_SCAN_INFO_CHECK", "0")
	t.Setenv("AI_DEFENSE_RETRY_COUNT_FOR_SCANNING", "2")

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"scan_status_info":{"scan_id":"scan-1","status":"RUNNING"}}`))
	})

	_, err := c.pollScanCompletion("scan-1")
	if err == nil {
		t.Fatal("expected poll timeout")
	}
	var pollErr *api.PollTimeoutError
	if !errors.As(err, &pollErr) {
		t.Fatalf("expected PollTimeoutError, got %T: %v", err, err)
	}
	if pollErr.ScanID != "scan-1" {
		t.Errorf("expected scan-1, got %q", pollErr.ScanID)
	}
}
