package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// BaseClient provides authenticated HTTP access to the AI Defense Management API.
type BaseClient struct {
	HTTPClient       *http.Client
	UploadHTTPClient *http.Client
	apiKey           string
	apiPrefix        string

	uploadFallback     *http.Client
	uploadFallbackOnce sync.Once
}

// NewBaseClient creates a BaseClient for the given API key and config.
func NewBaseClient(apiKey string, config *Config) *BaseClient {
	if config == nil {
		config = NewConfig("")
	}
	baseURL := strings.TrimRight(config.ManagementBaseURL, "/")
	apiVersion := config.APIVersion
	if apiVersion == "" {
		apiVersion = DefaultAPIVersion
	}
	apiPrefix := fmt.Sprintf("%s/%s/%s", baseURL, APIPrefix, apiVersion)

	timeout := config.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	uploadSecs := config.UploadTimeout
	if uploadSecs <= 0 {
		uploadSecs = timeout
	}
	uploadHTTPClient := newUploadHTTPClient(uploadSecs)

	return &BaseClient{
		HTTPClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		UploadHTTPClient: uploadHTTPClient,
		apiKey:           apiKey,
		apiPrefix:        apiPrefix,
	}
}

func (c *BaseClient) getURL(path string) string {
	path = strings.TrimLeft(path, "/")
	return fmt.Sprintf("%s/%s", c.apiPrefix, path)
}

// DoRequest performs an HTTP request with JSON body and returns the raw JSON response.
func (c *BaseClient) DoRequest(method, path string, params url.Values, body interface{}) (json.RawMessage, error) {
	reqURL := c.getURL(path)
	if len(params) > 0 {
		reqURL = reqURL + "?" + params.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, reqURL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set(AuthHeader, c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, &TransportError{Method: method, URL: reqURL, Cause: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &ApiError{
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
		}
		var errResp map[string]interface{}
		if json.Unmarshal(respBody, &errResp) == nil {
			if reqID, ok := errResp["request_id"].(string); ok {
				apiErr.RequestID = reqID
			}
			if msg, ok := errResp["message"].(string); ok {
				apiErr.Message = msg
			}
		}
		return nil, apiErr
	}

	if len(respBody) == 0 {
		return nil, nil
	}

	return json.RawMessage(respBody), nil
}

// DoRawPUT performs a raw PUT to a presigned upload URL returned by the API.
func (c *BaseClient) DoRawPUT(uploadURL string, body io.Reader, contentLength int64) error {
	if err := validateUploadURL(uploadURL); err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, uploadURL, body)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}
	req.ContentLength = contentLength

	resp, err := c.uploadHTTPClient().Do(req)
	if err != nil {
		return &TransportError{Method: http.MethodPut, URL: uploadURL, Cause: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return &ApiError{
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
		}
	}
	return nil
}

// Get performs a GET request.
func (c *BaseClient) Get(path string, params url.Values) (json.RawMessage, error) {
	return c.DoRequest(http.MethodGet, path, params, nil)
}

// Post performs a POST request with a JSON body.
func (c *BaseClient) Post(path string, body interface{}) (json.RawMessage, error) {
	return c.DoRequest(http.MethodPost, path, nil, body)
}

// PostWithParams performs a POST request with query parameters and a JSON body.
func (c *BaseClient) PostWithParams(path string, params url.Values, body interface{}) (json.RawMessage, error) {
	return c.DoRequest(http.MethodPost, path, params, body)
}

// Put performs a PUT request with a JSON body.
func (c *BaseClient) Put(path string, body interface{}) (json.RawMessage, error) {
	return c.DoRequest(http.MethodPut, path, nil, body)
}

// Delete performs a DELETE request.
func (c *BaseClient) Delete(path string) (json.RawMessage, error) {
	return c.DoRequest(http.MethodDelete, path, nil, nil)
}

// uploadHTTPClient returns the dedicated upload client, or a lazily initialized
// redirect-safe fallback when UploadHTTPClient was not configured.
func (c *BaseClient) uploadHTTPClient() *http.Client {
	if c.UploadHTTPClient != nil {
		return c.UploadHTTPClient
	}
	c.uploadFallbackOnce.Do(func() {
		secs := int(DefaultTimeout)
		if c.HTTPClient != nil && c.HTTPClient.Timeout > 0 {
			secs = int(c.HTTPClient.Timeout.Seconds())
		}
		c.uploadFallback = newUploadHTTPClient(secs)
	})
	return c.uploadFallback
}

func newUploadHTTPClient(uploadSecs int) *http.Client {
	return &http.Client{
		Timeout:       time.Duration(uploadSecs) * time.Second,
		CheckRedirect: uploadCheckRedirect,
	}
}

// uploadCheckRedirect re-validates each redirect target so a presigned URL cannot
// be redirected to a private or link-local host (SSRF).
func uploadCheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return fmt.Errorf("upload redirect limit exceeded")
	}
	return validateUploadURL(req.URL.String())
}

func validateUploadURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return &ValidationError{Message: "invalid presigned upload URL"}
	}

	host := u.Hostname()
	switch host {
	case "localhost", "127.0.0.1", "::1", "169.254.169.254":
		return &ValidationError{Message: "presigned upload URL targets a non-public host"}
	}
	if strings.HasPrefix(host, "169.254.") {
		return &ValidationError{Message: "presigned upload URL targets a link-local host"}
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
			return &ValidationError{Message: "presigned upload URL targets a non-public address"}
		}
	}
	return nil
}
