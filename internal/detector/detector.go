package detector

import (
	"bytes"
	"sort"
	"strings"
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
			var matchedSnd model.Event
			for len(pending) > 0 {
				snd := pending[0]
				if ev.Timestamp.Sub(snd.Timestamp) <= rule.MaxWait {
					pending = pending[1:]
					matched = true
					matchedSnd = snd
					break
				}
				// pending too old
				findings = append(findings, model.Finding{
					Timestamp: snd.Timestamp,
					Group:     ev.Group,
					Type:      model.FindingMissingResponse,
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
					Type:      model.FindingRcvFlood,
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
						Type:      model.FindingDuplicateRcv,
						Detail:    "duplicate rcv payload",
					})
					stats.DuplicateCount++
				}
			} else {
				duplicateCount = 1
				lastRcvPayload = ev.PayloadRaw
			}

			if matched && isSensorFault(ev, matchedSnd) {
				findings = append(findings, model.Finding{
					Timestamp: ev.Timestamp,
					Group:     ev.Group,
					Type:      model.FindingSensorFault,
					Detail:    "unexpected response payload after snd",
				})
			}
		}
	}

	for _, snd := range pending {
		findings = append(findings, model.Finding{
			Timestamp: snd.Timestamp,
			Group:     snd.Group,
			Type:      model.FindingMissingResponse,
			Detail:    "snd without rcv",
		})
		stats.MissingCount++
	}

	if float64(stats.RcvCount) > float64(stats.SndCount)*rule.ExcessRcvRatio {
		stats.Excessive = true
		findings = append(findings, model.Finding{
			Timestamp: lastTimestamp(events),
			Group:     groupName(events),
			Type:      model.FindingExcessRcv,
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

func isSensorFault(ev model.Event, snd model.Event) bool {
	if ev.Dir != "rcv" {
		return false
	}
	if snd.Dir != "snd" {
		return false
	}
	if !strings.HasPrefix(strings.ToUpper(ev.Group), "WLS") {
		return false
	}
	if len(ev.PayloadBytes) > 0 {
		return bytes.Count(ev.PayloadBytes, []byte{0}) == len(ev.PayloadBytes)
	}
	trimmed := strings.TrimSpace(strings.Trim(ev.PayloadRaw, "()"))
	if trimmed == "" {
		return false
	}
	for _, part := range strings.Split(trimmed, ",") {
		if strings.TrimSpace(part) != "00" && strings.TrimSpace(part) != "0" {
			return false
		}
	}
	return true
}
