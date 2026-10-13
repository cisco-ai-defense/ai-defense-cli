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
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	progressInterval = 100 * time.Millisecond
	progressBarWidth = 30
	bytesPerMiB      = 1 << 20
	spinnerInterval  = 100 * time.Millisecond
	fileSpinnerText  = "Upload complete. Waiting for scan status..."
	repoSpinnerText  = "Waiting for scan status..."
)

var (
	spinnerFrames = []string{"|", "/", "-", "\\"}
	spinnerClear  = "\r" + strings.Repeat(" ", 64) + "\r"
)

// uploadProgress draws the cisco-aidefense-sdk 2.2.0 upload bar on stderr.
type uploadProgress struct {
	w   io.Writer
	now func() time.Time

	mu       sync.Mutex
	lastDraw time.Time
	lineOpen bool // bar drawn without its final newline
	done     bool
}

func newUploadProgress() *uploadProgress {
	return &uploadProgress{w: os.Stderr, now: time.Now}
}

// report is the upload ProgressFunc. Bytes are reported as they are sent, so
// redraws are throttled; the final draw at 100% is never skipped.
func (p *uploadProgress) report(uploaded, total int64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.done {
		return
	}
	finished := total <= 0 || uploaded >= total
	now := p.now()
	if !finished && !p.lastDraw.IsZero() && now.Sub(p.lastDraw) < progressInterval {
		return
	}
	p.lastDraw = now
	fmt.Fprint(p.w, uploadLine(uploaded, total))
	if finished {
		fmt.Fprintln(p.w)
		p.done = true
		p.lineOpen = false
		return
	}
	p.lineOpen = true
}

// close ends an unfinished bar (failed upload) so later output starts on its
// own line. Safe to call more than once.
func (p *uploadProgress) close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.lineOpen {
		fmt.Fprintln(p.w)
		p.lineOpen = false
	}
	p.done = true
}

// uploadLine matches the SDK's _ConsoleUploadProgress output.
func uploadLine(uploaded, total int64) string {
	ratio := 1.0
	if total > 0 {
		ratio = min(1.0, float64(uploaded)/float64(total))
	}
	filled := int(progressBarWidth * ratio)
	bar := strings.Repeat("#", filled) + strings.Repeat("-", progressBarWidth-filled)
	return fmt.Sprintf("\rUploading [%s] %5.1f%% (%.1f/%.1f MiB)",
		bar, ratio*100, float64(uploaded)/bytesPerMiB, float64(total)/bytesPerMiB)
}

// statusSpinner draws the cisco-aidefense-sdk 2.2.0 polling spinner
// (_ConsoleStatusSpinner) on stderr while waiting for scan results. Repo scans
// upload nothing, so they use repoSpinnerText instead of the SDK's wording.
type statusSpinner struct {
	w    io.Writer
	text string

	mu      sync.Mutex
	frame   int
	drawn   bool
	stop    chan struct{}
	stopped chan struct{}
}

func newStatusSpinner(text string) *statusSpinner {
	return &statusSpinner{w: os.Stderr, text: text}
}

// start draws the first frame immediately, then redraws every 100 ms until
// close. Later calls are no-ops.
func (s *statusSpinner) start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stop != nil {
		return
	}
	s.stop = make(chan struct{})
	s.stopped = make(chan struct{})
	s.draw()
	go s.spin(s.stop, s.stopped)
}

func (s *statusSpinner) spin(stop <-chan struct{}, stopped chan<- struct{}) {
	defer close(stopped)
	ticker := time.NewTicker(spinnerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			s.mu.Lock()
			s.draw()
			s.mu.Unlock()
		}
	}
}

// draw must be called with s.mu held.
func (s *statusSpinner) draw() {
	fmt.Fprintf(s.w, "\r%s %s", spinnerFrames[s.frame%len(spinnerFrames)], s.text)
	s.frame++
	s.drawn = true
}

// close stops the spinner and clears its line. Safe to call more than once.
func (s *statusSpinner) close() {
	s.mu.Lock()
	stop, stopped := s.stop, s.stopped
	s.stop = nil
	s.mu.Unlock()
	if stop != nil {
		close(stop)
		<-stopped
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.drawn {
		fmt.Fprint(s.w, spinnerClear)
		s.drawn = false
	}
}
