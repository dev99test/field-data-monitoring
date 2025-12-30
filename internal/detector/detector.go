package detector

import (
	"sort"
	"time"

	"field-data-monitoring/internal/model"
	"field-data-monitoring/internal/rules"
)

// AnalyzeGroup detects anomalies for a set of events in a group.
func AnalyzeGroup(events []model.Event, rule rules.Rule) model.GroupResult {
	sort.Slice(events, func(i, j int) bool { return events[i].Timestamp.Before(events[j].Timestamp) })

	stats := model.Stats{Lines: len(events)}
	var findings []model.Finding

	var pending []model.Event
	consecutiveRcv := 0
	lastRcvPayload := ""
	duplicateCount := 1

	for _, ev := range events {
		if ev.Dir == "snd" {
			stats.SndCount++
			pending = append(pending, ev)
			consecutiveRcv = 0
			lastRcvPayload = ""
			duplicateCount = 1
			continue
		}

		if ev.Dir == "rcv" {
			stats.RcvCount++
			// Match with earliest pending
			matched := false
			for len(pending) > 0 {
				snd := pending[0]
				if ev.Timestamp.Sub(snd.Timestamp) <= rule.MaxWait {
					pending = pending[1:]
					matched = true
					break
				}
				// pending too old
				findings = append(findings, model.Finding{
					Timestamp: snd.Timestamp,
					Group:     ev.Group,
					Type:      "MissingResponse",
					Detail:    "snd without timely rcv",
				})
				stats.MissingCount++
				pending = pending[1:]
			}

			if !matched {
				consecutiveRcv++
			} else {
				consecutiveRcv = 0
			}

			if consecutiveRcv >= rule.RcvFloodThreshold {
				findings = append(findings, model.Finding{
					Timestamp: ev.Timestamp,
					Group:     ev.Group,
					Type:      "RcvFlood",
					Detail:    "rcv flood without snd",
				})
				stats.FloodCount++
			}

			if ev.PayloadRaw == lastRcvPayload {
				duplicateCount++
				if duplicateCount >= rule.DuplicateRcvRepeat {
					findings = append(findings, model.Finding{
						Timestamp: ev.Timestamp,
						Group:     ev.Group,
						Type:      "DuplicateRcv",
						Detail:    "duplicate rcv payload",
					})
					stats.DuplicateCount++
				}
			} else {
				duplicateCount = 1
				lastRcvPayload = ev.PayloadRaw
			}
		}
	}

	for _, snd := range pending {
		findings = append(findings, model.Finding{
			Timestamp: snd.Timestamp,
			Group:     snd.Group,
			Type:      "MissingResponse",
			Detail:    "snd without rcv",
		})
		stats.MissingCount++
	}

	if float64(stats.RcvCount) > float64(stats.SndCount)*rule.ExcessRcvRatio {
		stats.Excessive = true
		findings = append(findings, model.Finding{
			Timestamp: lastTimestamp(events),
			Group:     groupName(events),
			Type:      "ExcessiveResponse",
			Detail:    "rcv count exceeds ratio",
		})
	}

	return model.GroupResult{
		Group:    groupName(events),
		Stats:    stats,
		Findings: findings,
	}
}

func lastTimestamp(events []model.Event) time.Time {
	if len(events) == 0 {
		return time.Time{}
	}
	return events[len(events)-1].Timestamp
}

func groupName(events []model.Event) string {
	if len(events) == 0 {
		return ""
	}
	return events[0].Group
}
