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

package tui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// PrintBanner writes a titled panel (same box style as CLI --help) to w.
func PrintBanner(w io.Writer, title, body string) {
	if w == nil {
		w = os.Stdout
	}
	color := colorEnabled(w)
	total := terminalWidthFile(w)
	fmt.Fprint(w, panel(color, title, body, total)+"\n\n")
}

func colorEnabled(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("FORCE_COLOR") != "" || os.Getenv("CLICOLOR_FORCE") != "" {
		return true
	}
	if f, ok := w.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

func terminalWidthFile(w io.Writer) int {
	if f, ok := w.(*os.File); ok {
		if width, _, err := term.GetSize(int(f.Fd())); err == nil && width > 0 {
			return width
		}
	}
	return 80
}

const (
	cReset  = "\x1b[0m"
	cBorder = "\x1b[2m"
)

func paint(color bool, code, s string) string {
	if !color || s == "" {
		return s
	}
	return code + s + cReset
}

func panel(color bool, title, body string, total int) string {
	inner := total - 4
	if inner < 4 {
		inner = 4
	}

	var b strings.Builder
	prefix := "╭─ " + title + " "
	fill := total - displayWidth(prefix) - 1
	if fill < 0 {
		fill = 0
	}
	b.WriteString(paint(color, cBorder, prefix+strings.Repeat("─", fill)+"╮") + "\n")

	for _, line := range strings.Split(body, "\n") {
		gap := inner - displayWidth(line)
		if gap < 0 {
			gap = 0
		}
		b.WriteString(paint(color, cBorder, "│") + " " + line +
			strings.Repeat(" ", gap) + " " + paint(color, cBorder, "│") + "\n")
	}

	b.WriteString(paint(color, cBorder, "╰"+strings.Repeat("─", total-2)+"╯"))
	return b.String()
}

func displayWidth(s string) int {
	w, inEsc := 0, false
	for _, r := range s {
		switch {
		case inEsc:
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
		case r == 0x1b:
			inEsc = true
		default:
			w++
		}
	}
	return w
}
