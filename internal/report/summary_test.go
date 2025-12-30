package report

import (
	"strings"
	"testing"

	"field-data-monitoring/internal/model"
)

func TestBuildAndRenderSummary(t *testing.T) {
	findings := []model.Finding{
		{Group: "WLS1", Type: model.FindingMissingResponse},
		{Group: "WLS1", Type: model.FindingMissingResponse},
		{Group: "GATE1", Type: model.FindingSensorFault},
		{Group: "WLS1", Type: model.FindingExcessRcv},
	}
	groups := []model.GroupResult{{Group: "WLS1"}, {Group: "GATE1"}, {Group: "PUMP1"}}

	summary := BuildSummary(findings, groups)

	if summary["WLS1"][model.FindingMissingResponse] != 2 {
		t.Fatalf("expected two missing responses for WLS1, got %d", summary["WLS1"][model.FindingMissingResponse])
	}
	if summary["WLS1"][model.FindingSensorFault] != 0 {
		t.Fatalf("expected zero sensor faults for WLS1")
	}

	rendered := RenderSummary(summary)
	expectedBlock := "[WLS1]\n- 응답없음: 2\n- 센서고장: 0\n"
	if !strings.Contains(rendered, expectedBlock) {
		t.Fatalf("rendered summary missing expected block:\n%s", rendered)
	}
	if !strings.Contains(rendered, "[PUMP1]\n- 응답없음: 0") {
		t.Fatalf("rendered summary should include zero entries for PUMP1")
	}
}
