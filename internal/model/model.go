package model

import "time"

// Event represents a single log line.
type Event struct {
	Timestamp    time.Time
	Group        string
	Dir          string
	PayloadRaw   string
	PayloadBytes []byte
	KV           map[string]string
	File         string
	Line         int
}

// Finding is an anomaly detected for a group.
type Finding struct {
	Timestamp time.Time
	Group     string
	Type      string
	Detail    string
}

// Stats capture aggregated metrics per group.
type Stats struct {
	Lines          int
	InvalidLines   int
	SndCount       int
	RcvCount       int
	MissingCount   int
	FloodCount     int
	DuplicateCount int
	Excessive      bool
}

// GroupResult is the output per group.
type GroupResult struct {
	Group    string
	Stats    Stats
	Findings []Finding
}

// AnalysisResult aggregates all groups.
type AnalysisResult struct {
	Groups   []GroupResult
	Findings []Finding
}
