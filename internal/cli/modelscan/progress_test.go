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
	"strings"
	"sync"
	"testing"
	"time"
)

const mib = int64(1 << 20)

// syncBuffer lets the spinner goroutine and the test share one writer.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func newTestProgress(buf *bytes.Buffer, clock *time.Time) *uploadProgress {
	return &uploadProgress{w: buf, now: func() time.Time { return *clock }}
}

func TestUploadLine_MatchesSDK(t *testing.T) {
	tests := []struct {
		uploaded, total int64
		want            string
	}{
		{0, 4096 * mib, "\rUploading [------------------------------]   0.0% (0.0/4096.0 MiB)"},
		{1536 * mib, 4096 * mib, "\rUploading [###########-------------------]  37.5% (1536.0/4096.0 MiB)"},
		{4096 * mib, 4096 * mib, "\rUploading [##############################] 100.0% (4096.0/4096.0 MiB)"},
		{0, 0, "\rUploading [##############################] 100.0% (0.0/0.0 MiB)"},
	}
	for _, tt := range tests {
		if got := uploadLine(tt.uploaded, tt.total); got != tt.want {
			t.Errorf("uploadLine(%d, %d) =\n%q, want\n%q", tt.uploaded, tt.total, got, tt.want)
		}
	}
}

func TestUploadProgress_ThrottlesAndEndsWithNewline(t *testing.T) {
	var buf bytes.Buffer
	clock := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	p := newTestProgress(&buf, &clock)

	p.report(0, 100*mib)
	clock = clock.Add(50 * time.Millisecond)
	p.report(5*mib, 100*mib) // within 100 ms of the last draw: skipped
	clock = clock.Add(time.Second)
	p.report(40*mib, 100*mib)
	clock = clock.Add(10 * time.Millisecond)
	p.report(100*mib, 100*mib) // final draw is never throttled
	p.report(100*mib, 100*mib) // after completion: ignored
	p.close()                  // line already ended: writes nothing

	want := uploadLine(0, 100*mib) + uploadLine(40*mib, 100*mib) + uploadLine(100*mib, 100*mib) + "\n"
	if got := buf.String(); got != want {
		t.Fatalf("output =\n%q, want\n%q", got, want)
	}
}

func TestUploadProgress_CloseEndsUnfinishedBar(t *testing.T) {
	var buf bytes.Buffer
	clock := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	p := newTestProgress(&buf, &clock)

	p.report(10*mib, 100*mib)
	p.close()
	p.close()
	clock = clock.Add(time.Second)
	p.report(20*mib, 100*mib) // after close: ignored

	if got, want := buf.String(), uploadLine(10*mib, 100*mib)+"\n"; got != want {
		t.Fatalf("output =\n%q, want\n%q", got, want)
	}
}

func TestStatusSpinner_MatchesSDKAndClears(t *testing.T) {
	var buf syncBuffer
	s := &statusSpinner{w: &buf, text: fileSpinnerText}

	s.start()
	s.start() // second call must not start another spinner
	deadline := time.Now().Add(2 * time.Second)
	for strings.Count(buf.String(), fileSpinnerText) < 3 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	s.close()
	s.close()

	out := buf.String()
	if !strings.HasPrefix(out, "\r| Upload complete. Waiting for scan status...") {
		t.Fatalf("spinner must draw immediately, got %q", out)
	}
	for _, frame := range []string{"/", "-"} {
		if want := "\r" + frame + " Upload complete. Waiting for scan status..."; !strings.Contains(out, want) {
			t.Fatalf("frame %q missing from %q", want, out)
		}
	}
	if !strings.HasSuffix(out, "\r"+strings.Repeat(" ", 64)+"\r") || strings.Count(out, spinnerClear) != 1 {
		t.Fatalf("spinner must be cleared exactly once, got %q", out)
	}
}

func TestStatusSpinner_RepoTextOmitsUpload(t *testing.T) {
	var buf syncBuffer
	s := newStatusSpinner(repoSpinnerText)
	s.w = &buf

	s.start()
	s.close()

	out := buf.String()
	if !strings.HasPrefix(out, "\r| Waiting for scan status...") {
		t.Fatalf("got %q", out)
	}
	if strings.Contains(out, "Upload") {
		t.Fatalf("repo spinner must not mention an upload, got %q", out)
	}
}

func TestStatusSpinner_CloseWithoutStartWritesNothing(t *testing.T) {
	var buf syncBuffer
	(&statusSpinner{w: &buf}).close()
	if got := buf.String(); got != "" {
		t.Fatalf("output = %q, want empty", got)
	}
}

func TestUploadProgress_CloseWithoutOutputWritesNothing(t *testing.T) {
	var buf bytes.Buffer
	clock := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	newTestProgress(&buf, &clock).close()

	if got := buf.String(); got != "" {
		t.Fatalf("output = %q, want empty", got)
	}
}
