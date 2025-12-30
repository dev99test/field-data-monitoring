package report

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"field-data-monitoring/internal/model"
)

// PrintConsole outputs summary and findings to stdout.
func PrintConsole(result model.AnalysisResult) {
	for _, group := range result.Groups {
		s := group.Stats
		fmt.Printf("[%s] lines=%d invalid=%d snd=%d rcv=%d missing=%d flood=%d dup=%d excess=%v\n",
			group.Group, s.Lines, s.InvalidLines, s.SndCount, s.RcvCount, s.MissingCount, s.FloodCount, s.DuplicateCount, s.Excessive)
	}
	fmt.Println("Findings:")
	for _, f := range result.Findings {
		fmt.Printf("%s [%s] %s - %s\n", f.Timestamp.Format("2006-01-02 15:04:05.000"), f.Group, f.Type, f.Detail)
	}
}

// WriteJSON writes result as JSON to writer.
func WriteJSON(w io.Writer, result model.AnalysisResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

// SaveJSON writes result to file path.
func SaveJSON(path string, result model.AnalysisResult) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return WriteJSON(f, result)
}
