package output

import (
	"encoding/json"
	"fmt"
	"os"
)

// PrintJSON marshals v to indented JSON and prints it to stdout.
func PrintJSON(v interface{}) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
		return
	}
	fmt.Println(string(data))
}

// PrintRawJSON prints raw JSON bytes, re-indented for readability.
func PrintRawJSON(raw json.RawMessage) {
	if raw == nil {
		return
	}
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		fmt.Println(string(raw))
		return
	}
	PrintJSON(v)
}
