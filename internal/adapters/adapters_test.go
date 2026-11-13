package adapters

import (
	"testing"
)

func TestRejectInventedMappings(t *testing.T) {
	if _, e := (OTLP{}).Decode([]byte(`{"resourceLogs":[{"scopeLogs":[{"logRecords":[{"timeUnixNano":"1","attributes":[]}]}]}]}`)); e == nil {
		t.Fatal("guessed identity")
	}
	if _, e := (CloudWatch{}).Decode([]byte(`{"events":[{"eventId":"1","message":"unknown schema"}]}`)); e == nil {
		t.Fatal("accepted unknown native schema")
	}
	if _, e := (Normalized{}).Decode([]byte(`{"schema_version":1}`)); e == nil {
		t.Fatal("accepted missing fields")
	}
}
