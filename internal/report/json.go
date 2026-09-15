package report

import (
	"bytes"
	"encoding/json"
)

// RenderJSON writes the report as indented JSON. The output is deterministic:
// findings are already sorted by the findings package, and the per-group
// maps are emitted in a fixed key order.
func RenderJSON(rep *Report) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(rep); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
