package report

import (
	"fmt"
	"sort"
	"strings"

	"field-data-monitoring/internal/model"
)

var summaryOrder = []struct {
	Type  string
	Label string
}{
	{model.FindingMissingResponse, "응답없음"},
	{model.FindingExcessRcv, "응답과다"},
	{model.FindingRcvFlood, "응답폭주"},
	{model.FindingDuplicateRcv, "중복응답"},
	{model.FindingSensorFault, "센서고장"},
}

// BuildSummary aggregates findings by group and type. Groups provided are always included with zero counts.
func BuildSummary(findings []model.Finding, groups []model.GroupResult) map[string]map[string]int {
	summary := make(map[string]map[string]int)
	ensureGroup := func(group string) {
		if group == "" {
			return
		}
		if _, ok := summary[group]; !ok {
			summary[group] = make(map[string]int)
			for _, item := range summaryOrder {
				summary[group][item.Type] = 0
			}
		}
	}

	for _, g := range groups {
		ensureGroup(g.Group)
	}

	for _, f := range findings {
		normalized := normalizeType(f.Type)
		if normalized == "" {
			continue
		}
		ensureGroup(f.Group)
		summary[f.Group][normalized]++
	}

	return summary
}

// RenderSummary renders the summary into the fixed text block format.
func RenderSummary(summary map[string]map[string]int) string {
	var groups []string
	for g := range summary {
		groups = append(groups, g)
	}
	sort.Strings(groups)

	var b strings.Builder
	for i, g := range groups {
		b.WriteString(fmt.Sprintf("[%s]\n", g))
		counts := summary[g]
		for _, item := range summaryOrder {
			b.WriteString(fmt.Sprintf("- %s: %d\n", item.Label, counts[item.Type]))
		}
		if i < len(groups)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

func normalizeType(t string) string {
	upper := strings.ToUpper(t)
	switch upper {
        case model.FindingMissingResponse, "MISSINGRESPONSE":
                return model.FindingMissingResponse
        case model.FindingExcessRcv, "EXCESSIVERESPONSE":
                return model.FindingExcessRcv
        case model.FindingRcvFlood, "RCVFLOOD":
                return model.FindingRcvFlood
        case model.FindingDuplicateRcv, "DUPLICATERCV":
                return model.FindingDuplicateRcv
        case model.FindingSensorFault, "SENSORFAULT":
                return model.FindingSensorFault
	default:
		return ""
	}
}
