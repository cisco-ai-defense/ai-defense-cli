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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/cisco-ai-defense/ai-defense-cli/api"
)

const (
	testScanID   = "scan-1"
	testObjectID = "obj-1"
	testUploadID = "upload-1"
	testS3Host   = "model-bucket.s3.us-west-2.amazonaws.com"
)

type rewriteTransport struct{ target *url.URL }

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = t.target.Scheme
	req.URL.Host = t.target.Host
	return http.DefaultTransport.RoundTrip(req)
}

// fakeMultipartService emulates scan-manager plus the S3 presigned endpoints.
type fakeMultipartService struct {
	t        *testing.T
	partSize int64

	// omitMultipart makes create-object return only upload_url (old server).
	omitMultipart bool
	// dropPartURL omits this part number from part-urls responses.
	dropPartURL int
	// failPart, when set, decides the response for a part PUT attempt (1-based).
	failPart func(part, attempt int) (status int, body string, etag *string)
	// partURLHost overrides the host of returned part URLs.
	partURLHost string
	// abortStatus, when non-zero, is the abort endpoint's response status.
	abortStatus int
	// beforePut runs, without the lock held, before a part PUT is handled.
	beforePut func(part int)
	// scanRunning keeps the scan status RUNNING so polling times out.
	scanRunning bool
	// onPartURLs runs, without the lock held, around each part-urls request.
	onPartURLs func() (done func())

	mu             sync.Mutex
	createReqs     []CreateScanObjectRequest
	partURLBatch   [][]int
	attempts       map[int]int
	parts          map[int][]byte
	singlePut      []byte
	completeReq    *CompleteMultipartUploadRequest
	aborted        bool
	deleted        bool
	triggered      bool
	partPutsTotal  int
	fetchedParts   int
	maxOutstanding int
	requests       int
}

func (f *fakeMultipartService) handler(w http.ResponseWriter, r *http.Request) {
	if f.beforePut != nil && r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/part/") {
		n, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/part/"))
		f.beforePut(n)
	}
	if f.onPartURLs != nil && strings.HasSuffix(r.URL.Path, "/multipart/part-urls") {
		defer f.onPartURLs()()
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests++

	path := strings.TrimPrefix(r.URL.Path, "/api/ai-defense/v1/")
	objectPrefix := "scans/" + testScanID + "/objects/" + testObjectID + "/multipart/"
	switch {
	case r.Method == http.MethodPost && path == "scans/register":
		fmt.Fprintf(w, `{"scan_id":%q}`, testScanID)
	case r.Method == http.MethodPost && path == "scans/"+testScanID+"/objects":
		var req CreateScanObjectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			f.t.Errorf("decode create object: %v", err)
		}
		f.createReqs = append(f.createReqs, req)
		if !req.UseMultipartUpload || f.omitMultipart {
			fmt.Fprintf(w, `{"object_id":%q,"upload_url":"https://%s/single?X-Amz-Signature=secret"}`, testObjectID, testS3Host)
			return
		}
		fmt.Fprintf(w, `{"object_id":%q,"multipart_upload":{"upload_id":%q,"part_size_bytes":"%d"}}`, testObjectID, testUploadID, f.partSize)
	case r.Method == http.MethodPost && path == objectPrefix+"part-urls":
		var req GetMultipartUploadPartURLsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			f.t.Errorf("decode part-urls: %v", err)
		}
		if req.UploadID != testUploadID {
			f.t.Errorf("part-urls upload_id = %q", req.UploadID)
		}
		f.partURLBatch = append(f.partURLBatch, req.PartNumbers)
		f.fetchedParts += len(req.PartNumbers)
		f.maxOutstanding = max(f.maxOutstanding, f.fetchedParts-len(f.parts))
		host := testS3Host
		if f.partURLHost != "" {
			host = f.partURLHost
		}
		var parts []string
		for _, n := range req.PartNumbers {
			if n == f.dropPartURL {
				continue
			}
			parts = append(parts, fmt.Sprintf(`{"partNumber":%d,"uploadUrl":"https://%s/part/%d?X-Amz-Signature=secret"}`, n, host, n))
		}
		fmt.Fprintf(w, `{"parts":[%s]}`, strings.Join(parts, ","))
	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/part/"):
		n, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/part/"))
		body, _ := io.ReadAll(r.Body)
		f.partPutsTotal++
		f.attempts[n]++
		etag := fmt.Sprintf(`"etag-%d"`, n)
		if f.failPart != nil {
			if status, msg, override := f.failPart(n, f.attempts[n]); status != 0 {
				if status >= 300 && status < 400 {
					w.Header().Set("Location", "https://"+testS3Host+"/redirected")
				}
				w.WriteHeader(status)
				io.WriteString(w, msg)
				return
			} else if override != nil {
				etag = *override
			}
		}
		f.parts[n] = body
		if etag != "" {
			w.Header().Set("ETag", etag)
		}
	case r.Method == http.MethodPut && r.URL.Path == "/single":
		f.singlePut, _ = io.ReadAll(r.Body)
	case r.Method == http.MethodPost && path == objectPrefix+"complete":
		var req CompleteMultipartUploadRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			f.t.Errorf("decode complete: %v", err)
		}
		f.completeReq = &req
	case r.Method == http.MethodPost && path == objectPrefix+"abort":
		f.aborted = true
		if f.abortStatus != 0 {
			w.WriteHeader(f.abortStatus)
		}
	case r.Method == http.MethodPut && path == "scans/"+testScanID+"/run":
		f.triggered = true
	case r.Method == http.MethodGet && path == "scans/"+testScanID:
		status := "COMPLETED"
		if f.scanRunning {
			status = "RUNNING"
		}
		fmt.Fprintf(w, `{"scan_status_info":{"scan_id":%q,"status":%q}}`, testScanID, status)
	case r.Method == http.MethodDelete && path == "scans/"+testScanID:
		f.deleted = true
	default:
		f.t.Errorf("unexpected request: %s %s", r.Method, r.URL.String())
		w.WriteHeader(http.StatusNotFound)
	}
}

func newFakeMultipart(t *testing.T, partSize int64) (*fakeMultipartService, *Client) {
	t.Helper()
	f := &fakeMultipartService{t: t, partSize: partSize, attempts: map[int]int{}, parts: map[int][]byte{}}
	srv := httptest.NewServer(http.HandlerFunc(f.handler))
	t.Cleanup(srv.Close)
	target, _ := url.Parse(srv.URL)

	cfg := api.NewConfig("https://test.cisco.com")
	cfg.HTTPClient = &http.Client{Transport: &rewriteTransport{target: target}}

	prevBackoff := partRetryBackoff
	partRetryBackoff = time.Millisecond
	t.Cleanup(func() { partRetryBackoff = prevBackoff })

	return f, NewClient("test-key", cfg)
}

func writeTestFile(t *testing.T, size int) (string, []byte) {
	t.Helper()
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i * 7)
	}
	path := filepath.Join(t.TempDir(), "model.bin")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path, data
}

func (f *fakeMultipartService) reassemble() []byte {
	var buf bytes.Buffer
	for n := 1; n <= len(f.parts); n++ {
		buf.Write(f.parts[n])
	}
	return buf.Bytes()
}

func TestUploadFileWithOptions_MultipartRoundTrip(t *testing.T) {
	f, c := newFakeMultipart(t, 10)
	path, data := writeTestFile(t, 95) // 10 parts, last one 5 bytes

	var mu sync.Mutex
	var lastUploaded, lastTotal int64
	opts := UploadOptions{Concurrency: 3, Progress: func(uploaded, total int64) {
		mu.Lock()
		defer mu.Unlock()
		lastUploaded, lastTotal = uploaded, total
	}}

	if err := c.UploadFileWithOptions(testScanID, path, opts); err != nil {
		t.Fatalf("UploadFileWithOptions: %v", err)
	}

	if len(f.createReqs) != 1 || !f.createReqs[0].UseMultipartUpload || f.createReqs[0].Size != 95 {
		t.Fatalf("create request = %+v, want multipart with size 95", f.createReqs)
	}
	if !bytes.Equal(f.reassemble(), data) {
		t.Fatal("reassembled parts do not match the file")
	}
	if f.completeReq == nil || f.completeReq.UploadID != testUploadID || len(f.completeReq.Parts) != 10 {
		t.Fatalf("complete request = %+v", f.completeReq)
	}
	for i, p := range f.completeReq.Parts {
		if p.PartNumber != i+1 || p.ETag != fmt.Sprintf(`"etag-%d"`, i+1) {
			t.Fatalf("complete part %d = %+v, want ordered parts with verbatim ETags", i, p)
		}
	}
	if f.aborted {
		t.Fatal("abort must not be called on success")
	}
	if lastUploaded != 95 || lastTotal != 95 {
		t.Fatalf("final progress = %d/%d, want 95/95", lastUploaded, lastTotal)
	}
}

func TestUploadFileWithOptions_BatchesPartURLs(t *testing.T) {
	f, c := newFakeMultipart(t, 1)
	path, data := writeTestFile(t, 70) // 70 parts → batches 32, 32, 6

	if err := c.UploadFileWithOptions(testScanID, path, UploadOptions{}); err != nil {
		t.Fatalf("UploadFileWithOptions: %v", err)
	}
	if got := len(f.partURLBatch); got != 3 {
		t.Fatalf("part-urls calls = %d, want 3", got)
	}
	for _, batch := range f.partURLBatch {
		if len(batch) > partURLBatchSize {
			t.Fatalf("batch of %d exceeds %d", len(batch), partURLBatchSize)
		}
	}
	if !bytes.Equal(f.reassemble(), data) {
		t.Fatal("reassembled parts do not match the file")
	}
}

func TestUploadFileWithOptions_RetriesRecoverablePartErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"server error", http.StatusServiceUnavailable, "<Error><Code>SlowDown</Code></Error>"},
		{"throttled", http.StatusTooManyRequests, ""},
		{"expired URL", http.StatusForbidden, "<Error><Code>RequestExpired</Code><Message>Request has expired</Message></Error>"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, c := newFakeMultipart(t, 10)
			f.failPart = func(part, attempt int) (int, string, *string) {
				if part == 2 && attempt == 1 {
					return tt.status, tt.body, nil
				}
				return 0, "", nil
			}
			path, data := writeTestFile(t, 30)

			if err := c.UploadFileWithOptions(testScanID, path, UploadOptions{}); err != nil {
				t.Fatalf("UploadFileWithOptions: %v", err)
			}
			if f.attempts[2] != 2 {
				t.Fatalf("part 2 attempts = %d, want 2", f.attempts[2])
			}
			// One batch for all parts plus a single-part refresh for the retry.
			if got := len(f.partURLBatch); got != 2 || len(f.partURLBatch[1]) != 1 || f.partURLBatch[1][0] != 2 {
				t.Fatalf("part-urls calls = %v, want initial batch + refresh of part 2", f.partURLBatch)
			}
			if !bytes.Equal(f.reassemble(), data) {
				t.Fatal("reassembled parts do not match the file")
			}
		})
	}
}

func TestUploadFileWithOptions_GivesUpAfterMaxAttempts(t *testing.T) {
	f, c := newFakeMultipart(t, 10)
	f.failPart = func(part, attempt int) (int, string, *string) {
		if part == 1 {
			return http.StatusInternalServerError, "", nil
		}
		return 0, "", nil
	}
	path, _ := writeTestFile(t, 30)

	err := c.UploadFileWithOptions(testScanID, path, UploadOptions{Concurrency: 1})
	if err == nil {
		t.Fatal("expected upload error")
	}
	var apiErr *api.ApiError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusInternalServerError {
		t.Fatalf("error = %v, want wrapped 500 ApiError", err)
	}
	var uploadErr *api.UploadError
	if !errors.As(err, &uploadErr) || uploadErr.Part != 1 || !uploadErr.Transient {
		t.Fatalf("error = %#v, want transient UploadError for part 1", uploadErr)
	}
	if f.attempts[1] != maxPartUploadAttempts {
		t.Fatalf("part 1 attempts = %d, want %d", f.attempts[1], maxPartUploadAttempts)
	}
	if !f.aborted {
		t.Fatal("expected abort after exhausting retries")
	}
	if f.completeReq != nil {
		t.Fatal("complete must not be called after a failed part")
	}
}

func TestUploadFileWithOptions_NonRetryableFailures(t *testing.T) {
	empty := ""
	tests := []struct {
		name       string
		failPart   func(part, attempt int) (int, string, *string)
		wantCause  error
		wantStatus int
	}{
		{
			name: "bad request",
			failPart: func(part, attempt int) (int, string, *string) {
				return http.StatusBadRequest, "<Error><Code>InvalidArgument</Code><Message>bad</Message><AWSAccessKeyId>AKIAEXAMPLE</AWSAccessKeyId></Error>", nil
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "missing ETag",
			failPart: func(part, attempt int) (int, string, *string) {
				return 0, "", &empty
			},
			wantCause: errInvalidETag,
		},
		{
			name: "redirect is not followed",
			failPart: func(part, attempt int) (int, string, *string) {
				return http.StatusTemporaryRedirect, "", nil
			},
			wantStatus: http.StatusTemporaryRedirect,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, c := newFakeMultipart(t, 10)
			f.failPart = tt.failPart
			path, _ := writeTestFile(t, 30)

			err := c.UploadFileWithOptions(testScanID, path, UploadOptions{Concurrency: 1})
			if err == nil || err.Error() != "multipart upload part 1 failed" {
				t.Fatalf("error = %v, want SDK-style %q", err, "multipart upload part 1 failed")
			}
			if tt.wantCause != nil && !errors.Is(err, tt.wantCause) {
				t.Fatalf("error cause = %v, want %v", errors.Unwrap(err), tt.wantCause)
			}
			var uploadErr *api.UploadError
			if !errors.As(err, &uploadErr) || uploadErr.Part != 1 || uploadErr.Transient {
				t.Fatalf("error = %#v, want non-transient UploadError for part 1", uploadErr)
			}
			var apiErr *api.ApiError
			if tt.wantStatus != 0 && (!errors.As(err, &apiErr) || apiErr.StatusCode != tt.wantStatus) {
				t.Fatalf("error cause = %v, want ApiError %d kept for exit-code mapping", errors.Unwrap(err), tt.wantStatus)
			}
			if strings.Contains(err.Error(), "AKIAEXAMPLE") || strings.Contains(err.Error(), "secret") {
				t.Fatalf("error leaks S3 credentials or URL signature: %v", err)
			}
			for n, a := range f.attempts {
				if a != 1 {
					t.Fatalf("part %d attempts = %d, want no retry", n, a)
				}
			}
			if !f.aborted {
				t.Fatal("expected abort")
			}
		})
	}
}

func TestUploadFileWithOptions_MissingPartURLFails(t *testing.T) {
	f, c := newFakeMultipart(t, 10)
	f.dropPartURL = 2
	path, _ := writeTestFile(t, 30)

	err := c.UploadFileWithOptions(testScanID, path, UploadOptions{})
	if err == nil || !strings.Contains(err.Error(), "every requested multipart upload URL") {
		t.Fatalf("error = %v, want missing URL error", err)
	}
	if !f.aborted {
		t.Fatal("expected abort")
	}
}

func TestUploadFileWithOptions_TooManyPartsAborts(t *testing.T) {
	f, c := newFakeMultipart(t, 1)
	path, _ := writeTestFile(t, maxMultipartParts+1)

	err := c.UploadFileWithOptions(testScanID, path, UploadOptions{})
	if err == nil || !strings.Contains(err.Error(), "part count") {
		t.Fatalf("error = %v, want part count error", err)
	}
	if !f.aborted {
		t.Fatal("expected abort")
	}
	if f.partPutsTotal != 0 {
		t.Fatalf("uploaded %d parts, want none", f.partPutsTotal)
	}
}

func TestUploadFileWithOptions_SinglePutWhenDisabled(t *testing.T) {
	f, c := newFakeMultipart(t, 10)
	path, data := writeTestFile(t, 30)

	var progressed bool
	opts := UploadOptions{DisableMultipart: true, Progress: func(uploaded, total int64) { progressed = true }}
	if err := c.UploadFileWithOptions(testScanID, path, opts); err != nil {
		t.Fatalf("UploadFileWithOptions: %v", err)
	}
	if f.createReqs[0].UseMultipartUpload {
		t.Fatal("use_multipart_upload must not be sent when multipart is disabled")
	}
	if len(f.partURLBatch) != 0 || f.completeReq != nil {
		t.Fatal("multipart endpoints must not be called on the single-PUT path")
	}
	if !bytes.Equal(f.singlePut, data) {
		t.Fatalf("single PUT body = %d bytes, want %d", len(f.singlePut), len(data))
	}
	if !progressed {
		t.Fatal("expected progress callback on single PUT")
	}
}

func TestUploadFileWithOptions_ServerWithoutMultipartFails(t *testing.T) {
	f, c := newFakeMultipart(t, 10)
	f.omitMultipart = true
	path, _ := writeTestFile(t, 30)

	err := c.UploadFileWithOptions(testScanID, path, UploadOptions{})
	if !errors.Is(err, errMissingMultipartUpload) {
		t.Fatalf("error = %v, want %v", err, errMissingMultipartUpload)
	}
	if f.singlePut != nil {
		t.Fatal("must not fall back to a single PUT")
	}
}

func TestEmptyFileRejectedBeforeAnyRequest(t *testing.T) {
	f, c := newFakeMultipart(t, 10)
	path, _ := writeTestFile(t, 0)

	for _, opts := range []UploadOptions{{}, {DisableMultipart: true}} {
		if err := c.UploadFileWithOptions(testScanID, path, opts); !errors.Is(err, errEmptyFile) {
			t.Fatalf("UploadFileWithOptions(%+v) error = %v, want %v", opts, err, errEmptyFile)
		}
	}
	if _, err := c.ScanFileWithOptions(path, UploadOptions{}); !errors.Is(err, errEmptyFile) {
		t.Fatalf("ScanFileWithOptions error = %v, want %v", err, errEmptyFile)
	}
	if f.requests != 0 {
		t.Fatalf("made %d API requests, want none (no orphan scan)", f.requests)
	}
}

func TestUploadFileWithOptions_RejectsNonS3PartURL(t *testing.T) {
	f, c := newFakeMultipart(t, 10)
	f.partURLHost = "uploads.example.com"
	path, _ := writeTestFile(t, 30)

	err := c.UploadFileWithOptions(testScanID, path, UploadOptions{Concurrency: 1})
	if !errors.Is(err, errInvalidPartUploadURL) {
		t.Fatalf("error = %v, want %v", err, errInvalidPartUploadURL)
	}
	if f.partPutsTotal != 0 {
		t.Fatalf("uploaded %d parts to a non-S3 host, want none", f.partPutsTotal)
	}
	if !f.aborted {
		t.Fatal("expected abort")
	}
}

func TestUploadFileWithOptions_PrefetchesTwoURLBatches(t *testing.T) {
	f, c := newFakeMultipart(t, 1)
	// Hold part 1 until the second batch is fetched; with one worker this
	// only succeeds if URLs are fetched ahead of the in-flight part.
	f.beforePut = func(part int) {
		if part != 1 {
			return
		}
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			f.mu.Lock()
			n := len(f.partURLBatch)
			f.mu.Unlock()
			if n >= 2 {
				return
			}
			time.Sleep(time.Millisecond)
		}
		t.Error("second URL batch was not prefetched while part 1 was in flight")
	}
	path, data := writeTestFile(t, 100)

	if err := c.UploadFileWithOptions(testScanID, path, UploadOptions{Concurrency: 1}); err != nil {
		t.Fatalf("UploadFileWithOptions: %v", err)
	}
	if f.maxOutstanding > 2*partURLBatchSize {
		t.Fatalf("outstanding parts reached %d, want at most %d", f.maxOutstanding, 2*partURLBatchSize)
	}
	if !bytes.Equal(f.reassemble(), data) {
		t.Fatal("reassembled parts do not match the file")
	}
}

func TestUploadFileWithOptions_AbortFailureKeepsUploadError(t *testing.T) {
	f, c := newFakeMultipart(t, 10)
	f.abortStatus = http.StatusInternalServerError
	f.failPart = func(part, attempt int) (int, string, *string) { return http.StatusBadRequest, "", nil }
	path, _ := writeTestFile(t, 30)

	err := c.UploadFileWithOptions(testScanID, path, UploadOptions{Concurrency: 1})
	var uploadErr *api.UploadError
	if !errors.As(err, &uploadErr) || uploadErr.Part != 1 {
		t.Fatalf("error = %v, want the part 1 upload error", err)
	}
	if !f.aborted {
		t.Fatal("expected an abort attempt")
	}
}

func TestValidatePartUploadURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://bucket.s3.us-west-2.amazonaws.com/key?X-Amz-Signature=x", true},
		{"https://BUCKET.S3.AMAZONAWS.COM/key", true},
		{"https://bucket.s3.cn-north-1.amazonaws.com.cn/key", true},
		{"http://bucket.s3.amazonaws.com/key", false},
		{"https://uploads.example.com/key", false},
		{"https://amazonaws.com.evil.example/key", false},
		{"https://evilamazonaws.com/key", false},
		{"://bad", false},
	}
	for _, tt := range tests {
		if got := validatePartUploadURL(tt.url) == nil; got != tt.want {
			t.Errorf("validatePartUploadURL(%q) ok = %v, want %v", tt.url, got, tt.want)
		}
	}
}

func TestUploadFileWithOptions_InvalidConcurrency(t *testing.T) {
	f, c := newFakeMultipart(t, 10)
	path, _ := writeTestFile(t, 30)

	for _, n := range []int{-1, MaxUploadConcurrency + 1} {
		err := c.UploadFileWithOptions(testScanID, path, UploadOptions{Concurrency: n})
		var valErr *api.ValidationError
		if !errors.As(err, &valErr) {
			t.Fatalf("concurrency %d: error = %v, want ValidationError", n, err)
		}
	}
	if len(f.createReqs) != 0 {
		t.Fatal("no API call expected for invalid concurrency")
	}
}

func TestUploadOptions_ConcurrencyIgnoredWhenMultipartDisabled(t *testing.T) {
	t.Setenv(envWaitSecs, "0")
	t.Setenv(envRetryCount, "")
	for _, n := range []int{-1, 0, MaxUploadConcurrency + 1} {
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			f, c := newFakeMultipart(t, 10)
			path, data := writeTestFile(t, 30)
			opts := UploadOptions{DisableMultipart: true, Concurrency: n}

			if err := c.UploadFileWithOptions(testScanID, path, opts); err != nil {
				t.Fatalf("UploadFileWithOptions: %v", err)
			}
			if !bytes.Equal(f.singlePut, data) {
				t.Fatal("expected the whole file in a single PUT")
			}
			if _, err := c.ScanFileWithOptions(path, opts); err != nil {
				t.Fatalf("ScanFileWithOptions: %v", err)
			}
		})
	}
}

func TestUploadFileWithOptions_SinglePutFailureIsUploadError(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		wantTransient bool
	}{
		{"rejected", http.StatusForbidden, false},
		{"server error", http.StatusServiceUnavailable, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path := strings.TrimPrefix(r.URL.Path, "/api/ai-defense/v1/")
				switch {
				case r.Method == http.MethodPost && path == "scans/"+testScanID+"/objects":
					fmt.Fprintf(w, `{"object_id":%q,"upload_url":"https://%s/single"}`, testObjectID, testS3Host)
				case r.Method == http.MethodPut && r.URL.Path == "/single":
					w.WriteHeader(tt.status)
					io.WriteString(w, "<Error><Code>Denied</Code></Error>")
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
			}))
			t.Cleanup(srv.Close)
			target, _ := url.Parse(srv.URL)
			cfg := api.NewConfig("https://test.cisco.com")
			cfg.HTTPClient = &http.Client{Transport: &rewriteTransport{target: target}}
			c := NewClient("test-key", cfg)
			path, _ := writeTestFile(t, 30)

			err := c.UploadFileWithOptions(testScanID, path, UploadOptions{DisableMultipart: true})
			var uploadErr *api.UploadError
			if !errors.As(err, &uploadErr) || uploadErr.Part != 0 || uploadErr.Transient != tt.wantTransient {
				t.Fatalf("error = %v (%#v), want UploadError part 0 transient=%v", err, uploadErr, tt.wantTransient)
			}
		})
	}
}

func TestUploadFileWithOptions_SerializesURLRefreshes(t *testing.T) {
	f, c := newFakeMultipart(t, 1)
	const parts = partURLBatchSize
	f.failPart = func(part, attempt int) (int, string, *string) {
		if attempt == 1 {
			return http.StatusForbidden, "<Error><Code>ExpiredToken</Code></Error>", nil
		}
		return 0, "", nil
	}
	var inFlight, maxInFlight int32
	var refreshMu sync.Mutex
	f.onPartURLs = func() func() {
		refreshMu.Lock()
		inFlight++
		maxInFlight = max(maxInFlight, inFlight)
		refreshMu.Unlock()
		time.Sleep(2 * time.Millisecond) // widen the window for overlapping refreshes
		return func() {
			refreshMu.Lock()
			inFlight--
			refreshMu.Unlock()
		}
	}
	path, data := writeTestFile(t, parts)

	if err := c.UploadFileWithOptions(testScanID, path, UploadOptions{Concurrency: MaxUploadConcurrency}); err != nil {
		t.Fatalf("UploadFileWithOptions: %v", err)
	}
	if got := len(f.partURLBatch); got != 1+parts {
		t.Fatalf("part-urls calls = %d, want 1 batch + %d single-part refreshes", got, parts)
	}
	if maxInFlight != 1 {
		t.Fatalf("concurrent part-urls requests = %d, want refreshes serialized", maxInFlight)
	}
	if !bytes.Equal(f.reassemble(), data) {
		t.Fatal("reassembled parts do not match the file")
	}
}

func TestScanFileWithOptions_MultipartEndToEnd(t *testing.T) {
	t.Setenv("AI_DEFENSE_WAIT_TIME_SECS_SUCCESSIVE_SCAN_INFO_CHECK", "0")
	f, c := newFakeMultipart(t, 10)
	path, _ := writeTestFile(t, 25)

	raw, err := c.ScanFileWithOptions(path, UploadOptions{})
	if err != nil {
		t.Fatalf("ScanFileWithOptions: %v", err)
	}
	if !strings.Contains(string(raw), "COMPLETED") {
		t.Fatalf("scan result = %s", raw)
	}
	if f.completeReq == nil || !f.triggered {
		t.Fatal("expected multipart complete followed by scan trigger")
	}
	if f.deleted {
		t.Fatal("scan must not be deleted on success")
	}
}

func TestUploadFileWithOptions_ByteProgressRollsBackFailedAttempt(t *testing.T) {
	f, c := newFakeMultipart(t, 10)
	f.failPart = func(part, attempt int) (int, string, *string) {
		if part == 2 && attempt == 1 {
			return http.StatusServiceUnavailable, "", nil
		}
		return 0, "", nil
	}
	path, _ := writeTestFile(t, 25)

	var updates []int64
	opts := UploadOptions{
		Concurrency: 1,
		Progress:    func(uploaded, total int64) { updates = append(updates, uploaded) },
	}
	if err := c.UploadFileWithOptions(testScanID, path, opts); err != nil {
		t.Fatalf("UploadFileWithOptions: %v", err)
	}

	if updates[len(updates)-1] != 25 {
		t.Fatalf("final progress = %d, want 25 (updates %v)", updates[len(updates)-1], updates)
	}
	rolledBack := false
	for i, u := range updates {
		if u > 25 || u < 0 {
			t.Fatalf("progress out of range: %v", updates)
		}
		if i > 0 && u < updates[i-1] {
			rolledBack = true
		}
	}
	if !rolledBack {
		t.Fatalf("bytes sent by the failed attempt should be rolled back: %v", updates)
	}
}

func TestScanFileWithOptions_PollingStartedAfterTrigger(t *testing.T) {
	t.Setenv(envWaitSecs, "0")
	t.Setenv(envRetryCount, "")
	f, c := newFakeMultipart(t, 10)
	path, _ := writeTestFile(t, 25)

	var calls int
	var triggeredFirst bool
	c.SetEvents(ScanEvents{PollingStarted: func() {
		calls++
		f.mu.Lock()
		triggeredFirst = f.triggered
		f.mu.Unlock()
	}})
	if _, err := c.ScanFileWithOptions(path, UploadOptions{}); err != nil {
		t.Fatalf("ScanFileWithOptions: %v", err)
	}
	if calls != 1 || !triggeredFirst {
		t.Fatalf("PollingStarted calls = %d, after trigger = %v; want 1, true", calls, triggeredFirst)
	}
}

func TestScanFileWithOptions_PollTimeoutKeepsScan(t *testing.T) {
	t.Setenv(envWaitSecs, "0")
	t.Setenv(envRetryCount, "2")
	f, c := newFakeMultipart(t, 10)
	f.scanRunning = true
	path, _ := writeTestFile(t, 25)

	_, err := c.ScanFileWithOptions(path, UploadOptions{})
	var pollErr *api.PollTimeoutError
	if !errors.As(err, &pollErr) {
		t.Fatalf("error = %v, want PollTimeoutError", err)
	}
	if pollErr.StatusCommand != "cisco-aid-cli model-scan info --scan-id "+testScanID {
		t.Errorf("StatusCommand = %q", pollErr.StatusCommand)
	}
	if f.deleted {
		t.Fatal("scan must not be deleted on poll timeout")
	}
}

func TestScanFileWithOptions_CleansUpOnUploadFailure(t *testing.T) {
	f, c := newFakeMultipart(t, 10)
	f.failPart = func(part, attempt int) (int, string, *string) {
		return http.StatusBadRequest, "", nil
	}
	path, _ := writeTestFile(t, 25)

	if _, err := c.ScanFileWithOptions(path, UploadOptions{}); err == nil || !strings.Contains(err.Error(), "failed to upload file") {
		t.Fatalf("error = %v, want upload failure", err)
	}
	if !f.aborted || !f.deleted {
		t.Fatalf("aborted=%v deleted=%v, want both", f.aborted, f.deleted)
	}
	if f.triggered {
		t.Fatal("scan must not be triggered after an upload failure")
	}
}

func TestIsRetryablePartError(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{&api.TransportError{Method: "PUT", URL: "https://x", Cause: errors.New("reset")}, true},
		{&api.ApiError{StatusCode: 500}, true},
		{&api.ApiError{StatusCode: 429}, true},
		{&api.ApiError{StatusCode: 403, Message: "ExpiredToken: token expired"}, true},
		{&api.ApiError{StatusCode: 400, Message: "RequestExpired"}, true},
		{&api.ApiError{StatusCode: 403, Message: "AccessDenied"}, false},
		{&api.ApiError{StatusCode: 404}, false},
		{&api.ValidationError{Message: "invalid presigned upload URL"}, false},
		{errInvalidETag, false},
	}
	for _, tt := range tests {
		if got := isRetryablePartError(fmt.Errorf("wrapped: %w", tt.err)); got != tt.want {
			t.Errorf("isRetryablePartError(%v) = %v, want %v", tt.err, got, tt.want)
		}
	}
}

func TestCreateScanObjectResponse_MultipartCamelCase(t *testing.T) {
	var resp CreateScanObjectResponse
	if err := json.Unmarshal([]byte(`{"objectId":"o","multipartUpload":{"uploadId":"u","partSizeBytes":"67108864"}}`), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.MultipartUpload == nil || resp.MultipartUpload.UploadID != "u" || resp.MultipartUpload.PartSizeBytes != 64<<20 {
		t.Fatalf("multipart upload = %+v", resp.MultipartUpload)
	}
}

func TestNewClient_DefaultsUploadTimeout(t *testing.T) {
	c := NewClient("k", api.NewConfig("https://test.cisco.com"))
	if got := c.base.UploadHTTPClient.Timeout; got != DefaultUploadTimeoutSecs*time.Second {
		t.Fatalf("upload timeout = %v, want %ds", got, DefaultUploadTimeoutSecs)
	}

	cfg := api.NewConfig("https://test.cisco.com")
	cfg.UploadTimeout = 42
	c = NewClient("k", cfg)
	if got := c.base.UploadHTTPClient.Timeout; got != 42*time.Second {
		t.Fatalf("explicit upload timeout = %v, want 42s", got)
	}
	if cfg.UploadTimeout != 42 {
		t.Fatal("NewClient must not mutate the caller's config")
	}
}
