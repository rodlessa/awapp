package main

import (
	"encoding/json"
	"testing"
)

// FuzzConfigJSON makes sure no config JSON can crash the loader or the
// validation warnings (they must be pure best-effort).
func FuzzConfigJSON(f *testing.F) {
	f.Add([]byte(`{"api_key":"k","city":"Fortaleza","size":15,"stars":"light","moon":"auto","provider":"auto","theme":"default"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"size":999,"stars":"banana","moon":"sometimes","provider":"nope","theme":"nope"}`))
	f.Add([]byte(`{"unknown_key":1}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		var raw map[string]json.RawMessage
		_ = json.Unmarshal(data, &raw)
		var fc fileConfig
		_ = json.Unmarshal(data, &fc)
		warnConfig(fc, "fuzz-config.json") // must not panic
	})
}
