package detector

import (
	"testing"
	"time"

	"field-data-monitoring/internal/model"
	"field-data-monitoring/internal/rules"
)

func TestAnalyzeGroupMissingAndDuplicate(t *testing.T) {
	events := []model.Event{
		{Timestamp: time.Date(2025, 12, 30, 14, 10, 5, 0, time.UTC), Group: "WLS1", Dir: "snd", PayloadRaw: "(AA)"},
		{Timestamp: time.Date(2025, 12, 30, 14, 10, 10, 0, time.UTC), Group: "WLS1", Dir: "rcv", PayloadRaw: "(AA)"},
		{Timestamp: time.Date(2025, 12, 30, 14, 10, 10, 100000000, time.UTC), Group: "WLS1", Dir: "rcv", PayloadRaw: "(AA)"},
		{Timestamp: time.Date(2025, 12, 30, 14, 10, 10, 200000000, time.UTC), Group: "WLS1", Dir: "rcv", PayloadRaw: "(AA)"},
	}
	rule := rules.Rule{MaxWait: 3 * time.Second, ExcessRcvRatio: 1.5, RcvFloodThreshold: 2, DuplicateRcvRepeat: 2}
	res := AnalyzeGroup(events, rule)
	if res.Stats.MissingCount == 0 {
		t.Fatalf("expected missing response")
	}
	if res.Stats.DuplicateCount == 0 {
		t.Fatalf("expected duplicate detection")
	}
	if !res.Stats.Excessive {
		t.Fatalf("expected excessive response flag")
	}
}

func TestSensorFaultDetection(t *testing.T) {
	events := []model.Event{
		{Timestamp: time.Date(2025, 12, 30, 14, 10, 5, 0, time.UTC), Group: "WLS2", Dir: "rcv", PayloadRaw: "(00)", PayloadBytes: []byte{0}},
	}
	rule := rules.Rule{MaxWait: 3 * time.Second, ExcessRcvRatio: 1.5, RcvFloodThreshold: 3, DuplicateRcvRepeat: 2}
	res := AnalyzeGroup(events, rule)
	if len(res.Findings) == 0 {
		t.Fatalf("expected sensor fault finding")
	}
	if res.Findings[0].Type != model.FindingSensorFault {
		t.Fatalf("expected sensor fault type, got %s", res.Findings[0].Type)
	}
}
