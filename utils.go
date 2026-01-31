package logging

import "encoding/json"

// jsonMarshal marshals data to JSON, fallback on error
func jsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}