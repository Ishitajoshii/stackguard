package adapters

import "testing"

func FuzzAdapterValidation(f *testing.F) {
	f.Add([]byte(`{"schema_version":1,"id":"fixture","ts":"2026-01-01T00:00:00Z","source":"algorithm","session":"s","span":"p","agent":"a","kind":"tool_call"}`))
	f.Add([]byte(`{"resourceLogs":[]}`))
	registry := DefaultRegistry()
	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > 1<<20 {
			return
		}
		for _, adapter := range registry {
			events, err := adapter.Decode(b)
			if err != nil {
				continue
			}
			for _, event := range events {
				if err := event.Validate(); err != nil {
					t.Fatal("adapter emitted an invalid event", err)
				}
			}
		}
	})
}
