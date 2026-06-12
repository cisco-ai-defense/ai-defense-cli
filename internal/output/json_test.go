package output

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestPrintJSON_Map(t *testing.T) {
	got := captureStdout(t, func() {
		PrintJSON(map[string]string{"key": "value"})
	})

	if !strings.Contains(got, `"key": "value"`) {
		t.Errorf("expected JSON with key/value, got %q", got)
	}
}

func TestPrintJSON_Nil(t *testing.T) {
	got := captureStdout(t, func() {
		PrintJSON(nil)
	})
	if strings.TrimSpace(got) != "null" {
		t.Errorf("expected 'null', got %q", got)
	}
}

func TestPrintJSON_Indented(t *testing.T) {
	got := captureStdout(t, func() {
		PrintJSON(map[string]interface{}{"a": 1, "b": "two"})
	})
	if !strings.Contains(got, "  ") {
		t.Error("expected indented output")
	}
}

func TestPrintRawJSON_ValidJSON(t *testing.T) {
	raw := json.RawMessage(`{"status":"ok"}`)
	got := captureStdout(t, func() {
		PrintRawJSON(raw)
	})
	if !strings.Contains(got, `"status": "ok"`) {
		t.Errorf("expected re-indented JSON, got %q", got)
	}
}

func TestPrintRawJSON_Nil(t *testing.T) {
	got := captureStdout(t, func() {
		PrintRawJSON(nil)
	})
	if got != "" {
		t.Errorf("expected empty output for nil, got %q", got)
	}
}

func TestPrintRawJSON_InvalidJSON(t *testing.T) {
	raw := json.RawMessage(`not json`)
	got := captureStdout(t, func() {
		PrintRawJSON(raw)
	})
	if !strings.Contains(got, "not json") {
		t.Errorf("expected raw passthrough for invalid JSON, got %q", got)
	}
}
