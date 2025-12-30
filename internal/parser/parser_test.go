package parser

import (
	"testing"
	"time"
)

func TestParseLineBytes(t *testing.T) {
	line := "2025-12-30 14:10:05.211 snd: (FA, FF, 07)"
	ev, err := ParseLine(line, "file", "WLS1", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.Dir != "snd" || ev.Group != "WLS1" {
		t.Fatalf("unexpected fields")
	}
	if len(ev.PayloadBytes) != 3 {
		t.Fatalf("expected 3 bytes, got %d", len(ev.PayloadBytes))
	}
}

func TestParseLineKV(t *testing.T) {
	line := "2025-11-21 22:58:23.253 rcv: GATE=DOWN OK,DETECTOR=ERROR"
	ev, err := ParseLine(line, "file", "GATE1", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ev.KV["GATE"] != "DOWN OK" {
		t.Fatalf("kv parse failed")
	}
}

func TestParseFile(t *testing.T) {
	since := time.Date(2025, 12, 30, 14, 10, 6, 0, time.UTC)
	events, invalid, err := ParseFile("../../testdata/log/WLS1/sample.log", "WLS1", &since, nil)
	if err != nil {
		t.Fatalf("parse file error: %v", err)
	}
	if invalid != 0 {
		t.Fatalf("expected no invalid lines")
	}
	for _, ev := range events {
		if ev.Timestamp.Before(since) {
			t.Fatalf("filter failed")
		}
	}
}
