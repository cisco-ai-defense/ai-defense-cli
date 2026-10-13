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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cisco-ai-defense/ai-defense-cli/api"
)

const (
	// DefaultUploadConcurrency is the number of multipart parts uploaded in parallel.
	DefaultUploadConcurrency = 10
	// MaxUploadConcurrency is the largest accepted UploadOptions.Concurrency.
	MaxUploadConcurrency = 32

	maxMultipartParts     = 10_000
	partURLBatchSize      = 32
	maxPartUploadAttempts = 3
	maxETagLength         = 2048
)

// partRetryBackoff is the delay before the first part retry; it doubles per attempt.
var partRetryBackoff = 500 * time.Millisecond

var presignedURLExpiryCodes = []string{"ExpiredToken", "RequestExpired"}

// partUploadHostSuffixes are the only hosts multipart part URLs may target.
var partUploadHostSuffixes = []string{".amazonaws.com", ".amazonaws.com.cn"}

var (
	errInvalidETag            = errors.New("part upload did not return a valid ETag")
	errInvalidPartUploadURL   = errors.New("service returned an invalid multipart upload URL")
	errMissingMultipartUpload = errors.New("scan object response did not include multipart upload details")
	errEmptyFile              = errors.New("file must not be empty")
)

// ProgressFunc receives the cumulative uploaded bytes and the total file size.
type ProgressFunc func(uploaded, total int64)

// UploadOptions controls how a model file is uploaded for scanning.
type UploadOptions struct {
	// DisableMultipart forces a single presigned PUT of the whole file.
	DisableMultipart bool
	// Concurrency is the number of parts uploaded in parallel (0 = default).
	// Ignored, and not validated, when DisableMultipart is set.
	Concurrency int
	// Progress, if set, is called as bytes are uploaded. Calls are serialized.
	Progress ProgressFunc
}

// ValidateUploadConcurrency reports whether n is an accepted upload concurrency.
func ValidateUploadConcurrency(n int) error {
	if n < 1 || n > MaxUploadConcurrency {
		return &api.ValidationError{
			Message: fmt.Sprintf("upload concurrency must be between 1 and %d, got %d", MaxUploadConcurrency, n),
		}
	}
	return nil
}

// concurrency resolves the multipart concurrency; it returns 0 and no error
// when multipart is disabled.
func (o UploadOptions) concurrency() (int, error) {
	if o.DisableMultipart {
		return 0, nil
	}
	if o.Concurrency == 0 {
		return DefaultUploadConcurrency, nil
	}
	if err := ValidateUploadConcurrency(o.Concurrency); err != nil {
		return 0, err
	}
	return o.Concurrency, nil
}

// GetMultipartUploadPartURLs requests presigned URLs for one batch of parts.
func (c *Client) GetMultipartUploadPartURLs(scanID, objectID string, req *GetMultipartUploadPartURLsRequest) (*GetMultipartUploadPartURLsResponse, error) {
	resp, err := c.base.Post(MultipartPartURLs(scanID, objectID), req)
	if err != nil {
		return nil, err
	}
	var result GetMultipartUploadPartURLsResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse multipart part URLs response: %w", err)
	}
	return &result, nil
}

// CompleteMultipartUpload finalizes a multipart upload once every part has an ETag.
func (c *Client) CompleteMultipartUpload(scanID, objectID string, req *CompleteMultipartUploadRequest) error {
	_, err := c.base.Post(MultipartComplete(scanID, objectID), req)
	return err
}

// AbortMultipartUpload releases an incomplete multipart upload.
func (c *Client) AbortMultipartUpload(scanID, objectID string, req *AbortMultipartUploadRequest) error {
	_, err := c.base.Post(MultipartAbort(scanID, objectID), req)
	return err
}

// UploadFileWithOptions creates the scan object for filePath and uploads the
// file with S3 multipart upload, or a single presigned PUT when
// DisableMultipart is set.
func (c *Client) UploadFileWithOptions(scanID, filePath string, opts UploadOptions) error {
	concurrency, err := opts.concurrency()
	if err != nil {
		return err
	}
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}
	size := info.Size()
	if size == 0 {
		return errEmptyFile
	}

	req := &CreateScanObjectRequest{
		FileName:           info.Name(),
		Size:               size,
		UseMultipartUpload: !opts.DisableMultipart,
	}
	obj, err := c.createScanObject(scanID, req)
	if err != nil {
		return fmt.Errorf("failed to create scan object: %w", err)
	}

	if !req.UseMultipartUpload {
		if obj.UploadURL == "" {
			return fmt.Errorf("scan object response did not include an upload URL")
		}
		return c.uploadWholeFile(obj.UploadURL, filePath, size, opts.Progress)
	}
	if obj.MultipartUpload == nil {
		return errMissingMultipartUpload
	}

	up := &multipartUpload{
		client:      c,
		scanID:      scanID,
		objectID:    obj.ObjectID,
		uploadID:    obj.MultipartUpload.UploadID,
		partSize:    int64(obj.MultipartUpload.PartSizeBytes),
		size:        size,
		concurrency: concurrency,
		progress:    opts.Progress,
	}
	if err := up.run(filePath); err != nil {
		up.abort()
		return err
	}
	return nil
}

func (c *Client) uploadWholeFile(uploadURL, filePath string, size int64, progress ProgressFunc) error {
	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer f.Close()

	var body io.Reader = f
	if progress != nil {
		progress(0, size)
		body = &progressReader{r: f, total: size, progress: progress}
	}
	if err := c.base.DoRawPUT(uploadURL, body, size); err != nil {
		return asUploadError(0, err)
	}
	return nil
}

// asUploadError marks object-store responses and transport failures as
// UploadError; other errors (e.g. URL validation) are returned unchanged.
func asUploadError(part int, err error) error {
	var apiErr *api.ApiError
	var transportErr *api.TransportError
	if errors.As(err, &apiErr) || errors.As(err, &transportErr) || part > 0 {
		return &api.UploadError{Part: part, Transient: isRetryablePartError(err), Cause: err}
	}
	return err
}

type progressReader struct {
	r        io.Reader
	read     int64
	total    int64
	progress ProgressFunc
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.read += int64(n)
		p.progress(p.read, p.total)
	}
	return n, err
}

type partJob struct {
	number int
	url    string
}

type multipartUpload struct {
	client      *Client
	scanID      string
	objectID    string
	uploadID    string
	partSize    int64
	size        int64
	concurrency int
	progress    ProgressFunc

	file *os.File

	// refreshMu serializes single-part URL refreshes on retry (as the SDK's
	// refresh lock does), so a batch of expired URLs does not burst the API.
	refreshMu sync.Mutex

	progressMu sync.Mutex
	uploaded   int64
}

func (m *multipartUpload) run(filePath string) error {
	if m.uploadID == "" || m.partSize <= 0 {
		return fmt.Errorf("service returned invalid multipart upload details")
	}
	partCount := (m.size + m.partSize - 1) / m.partSize
	if partCount < 1 || partCount > maxMultipartParts {
		return fmt.Errorf("invalid multipart upload part count %d (allowed 1-%d)", partCount, maxMultipartParts)
	}

	f, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer f.Close()
	m.file = f

	etags, err := m.uploadParts(int(partCount))
	if err != nil {
		return err
	}

	parts := make([]CompletedMultipartUploadPart, len(etags))
	for i, etag := range etags {
		parts[i] = CompletedMultipartUploadPart{PartNumber: i + 1, ETag: etag}
	}
	if err := m.client.CompleteMultipartUpload(m.scanID, m.objectID, &CompleteMultipartUploadRequest{
		UploadID: m.uploadID,
		Parts:    parts,
	}); err != nil {
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}
	return nil
}

// abort is best effort, like CleanupScanData: the original upload error is
// what the caller reports.
func (m *multipartUpload) abort() {
	_ = m.client.AbortMultipartUpload(m.scanID, m.objectID, &AbortMultipartUploadRequest{UploadID: m.uploadID})
}

// uploadParts uploads every part and returns their ETags indexed by part number - 1.
// At most two URL batches of parts are outstanding (fetched but not finished);
// the next batch is fetched once a full batch of slots frees up, so workers
// can cross batch boundaries without waiting for the slowest part.
func (m *multipartUpload) uploadParts(partCount int) ([]string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		failOnce sync.Once
		firstErr error
	)
	fail := func(err error) {
		failOnce.Do(func() {
			firstErr = err
			cancel()
		})
	}

	m.reportProgress(0)

	const maxOutstanding = 2 * partURLBatchSize
	slots := make(chan struct{}, maxOutstanding)
	jobs := make(chan partJob, maxOutstanding)
	go func() {
		defer close(jobs)
		for start := 1; start <= partCount; start += partURLBatchSize {
			end := min(start+partURLBatchSize-1, partCount)
			for n := start; n <= end; n++ {
				select {
				case slots <- struct{}{}:
				case <-ctx.Done():
					return
				}
			}
			urls, err := m.partURLs(start, end)
			if err != nil {
				fail(err)
				return
			}
			for n := start; n <= end; n++ {
				select {
				case jobs <- partJob{number: n, url: urls[n]}:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	etags := make([]string, partCount)
	var wg sync.WaitGroup
	for range min(m.concurrency, partCount) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				if ctx.Err() == nil {
					if etag, err := m.uploadPart(ctx, job); err != nil {
						fail(err)
					} else {
						etags[job.number-1] = etag
					}
				}
				<-slots
			}
		}()
	}
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}
	return etags, nil
}

// partURLs fetches presigned URLs for parts start..end and requires the
// response to contain exactly those part numbers.
func (m *multipartUpload) partURLs(start, end int) (map[int]string, error) {
	numbers := make([]int, 0, end-start+1)
	for n := start; n <= end; n++ {
		numbers = append(numbers, n)
	}
	resp, err := m.client.GetMultipartUploadPartURLs(m.scanID, m.objectID, &GetMultipartUploadPartURLsRequest{
		UploadID:    m.uploadID,
		PartNumbers: numbers,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get multipart upload URLs: %w", err)
	}
	urls := make(map[int]string, len(resp.Parts))
	for _, p := range resp.Parts {
		if p.PartNumber >= start && p.PartNumber <= end && p.UploadURL != "" {
			urls[p.PartNumber] = p.UploadURL
		}
	}
	if len(resp.Parts) != len(numbers) || len(urls) != len(numbers) {
		return nil, fmt.Errorf("service did not return every requested multipart upload URL")
	}
	return urls, nil
}

func (m *multipartUpload) uploadPart(ctx context.Context, job partJob) (string, error) {
	uploadURL := job.url
	for attempt := 0; ; attempt++ {
		etag, err := m.putPart(ctx, job.number, uploadURL)
		if err == nil {
			return etag, nil
		}
		if attempt+1 >= maxPartUploadAttempts || !isRetryablePartError(err) || ctx.Err() != nil {
			return "", asUploadError(job.number, err)
		}
		select {
		case <-time.After(partRetryBackoff << attempt):
		case <-ctx.Done():
			return "", ctx.Err()
		}
		if uploadURL, err = m.refreshPartURL(ctx, job.number); err != nil {
			return "", err
		}
	}
}

// refreshPartURL fetches a fresh URL for one part, one refresh at a time.
func (m *multipartUpload) refreshPartURL(ctx context.Context, number int) (string, error) {
	m.refreshMu.Lock()
	defer m.refreshMu.Unlock()
	if err := ctx.Err(); err != nil {
		return "", err
	}
	urls, err := m.partURLs(number, number)
	if err != nil {
		return "", err
	}
	return urls[number], nil
}

func (m *multipartUpload) putPart(ctx context.Context, number int, uploadURL string) (string, error) {
	if err := validatePartUploadURL(uploadURL); err != nil {
		return "", err
	}
	offset := int64(number-1) * m.partSize
	length := m.partLength(number)
	var body io.Reader = io.NewSectionReader(m.file, offset, length)
	var counter *partCounter
	if m.progress != nil {
		counter = &partCounter{r: body, report: m.reportProgress}
		body = counter
	}
	header, err := m.client.base.DoRawPUTNoRedirect(ctx, uploadURL, body, length)
	if err == nil {
		if etag := header.Get("ETag"); etag == "" || len(etag) > maxETagLength {
			err = errInvalidETag
		}
	}
	if counter != nil {
		sent := counter.stop()
		if err != nil {
			m.reportProgress(-sent)
		} else {
			m.reportProgress(length - sent)
		}
	}
	if err != nil {
		return "", err
	}
	return header.Get("ETag"), nil
}

// partCounter reports bytes as the HTTP client reads a part body. The
// transport may still read after the request returns, so stop freezes the
// count before the caller rolls it back or tops it up to the part length.
type partCounter struct {
	r       io.Reader
	report  func(delta int64)
	mu      sync.Mutex
	sent    int64
	stopped bool
}

func (p *partCounter) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.mu.Lock()
		if !p.stopped {
			p.sent += int64(n)
			p.report(int64(n))
		}
		p.mu.Unlock()
	}
	return n, err
}

func (p *partCounter) stop() int64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopped = true
	return p.sent
}

// validatePartUploadURL requires an HTTPS AWS S3 host for multipart part URLs.
func validatePartUploadURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" {
		return errInvalidPartUploadURL
	}
	host := strings.ToLower(u.Hostname())
	for _, suffix := range partUploadHostSuffixes {
		if strings.HasSuffix(host, suffix) {
			return nil
		}
	}
	return errInvalidPartUploadURL
}

func (m *multipartUpload) partLength(number int) int64 {
	offset := int64(number-1) * m.partSize
	return min(m.partSize, m.size-offset)
}

func (m *multipartUpload) reportProgress(delta int64) {
	if m.progress == nil {
		return
	}
	m.progressMu.Lock()
	defer m.progressMu.Unlock()
	m.uploaded += delta
	m.progress(m.uploaded, m.size)
}

// isRetryablePartError reports whether a part upload failure is transient:
// network errors, throttling, server errors, or an expired presigned URL.
func isRetryablePartError(err error) bool {
	var transportErr *api.TransportError
	if errors.As(err, &transportErr) {
		return true
	}
	var apiErr *api.ApiError
	if !errors.As(err, &apiErr) {
		return false
	}
	if apiErr.StatusCode == http.StatusTooManyRequests || apiErr.StatusCode >= 500 {
		return true
	}
	if apiErr.StatusCode == http.StatusBadRequest || apiErr.StatusCode == http.StatusForbidden {
		for _, code := range presignedURLExpiryCodes {
			if strings.Contains(apiErr.Message, code) {
				return true
			}
		}
	}
	return false
}
