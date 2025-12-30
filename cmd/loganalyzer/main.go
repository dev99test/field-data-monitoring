package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"field-data-monitoring/internal/detector"
	"field-data-monitoring/internal/model"
	"field-data-monitoring/internal/parser"
	"field-data-monitoring/internal/report"
	"field-data-monitoring/internal/rules"
)

func main() {
	root := flag.String("root", "", "log root directory")
	configPath := flag.String("config", "configs/rules.yaml", "rules yaml path")
	jsonOut := flag.Bool("json", false, "print findings as json")
	outFile := flag.String("out", "", "save json result to file")
	summaryOnly := flag.Bool("summary-only", false, "print anomaly counts per group without details")
	sinceStr := flag.String("since", "", "only analyze events since timestamp")
	untilStr := flag.String("until", "", "only analyze events until timestamp")
	flag.Parse()

	if *root == "" {
		fmt.Println("--root is required")
		os.Exit(1)
	}

	var sincePtr, untilPtr *time.Time
	if *sinceStr != "" {
		t, err := time.Parse("2006-01-02 15:04:05.000", *sinceStr)
		if err != nil {
			fmt.Println("invalid --since format")
			os.Exit(1)
		}
		sincePtr = &t
	}
	if *untilStr != "" {
		t, err := time.Parse("2006-01-02 15:04:05.000", *untilStr)
		if err != nil {
			fmt.Println("invalid --until format")
			os.Exit(1)
		}
		untilPtr = &t
	}

	cfg, err := rules.Load(*configPath)
	if err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	result, err := analyzeRoot(*root, cfg, sincePtr, untilPtr)
	if err != nil {
		fmt.Printf("analyze failed: %v\n", err)
		os.Exit(1)
	}

	if *summaryOnly {
		summary := report.BuildSummary(result.Findings, result.Groups)
		fmt.Print(report.RenderSummary(summary))
	} else if *jsonOut {
		if err := report.WriteJSON(os.Stdout, result); err != nil {
			fmt.Printf("failed to render json: %v\n", err)
		}
	} else {
		report.PrintConsole(result)
	}

	if *outFile != "" {
		if err := report.SaveJSON(*outFile, result); err != nil {
			fmt.Printf("failed to save json: %v\n", err)
		}
	}
}

func analyzeRoot(root string, cfg rules.Config, since, until *time.Time) (model.AnalysisResult, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return model.AnalysisResult{}, err
	}

	var groupResults []model.GroupResult
	var allFindings []model.Finding

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		group := entry.Name()
		groupPath := filepath.Join(root, group)
		events, invalid, err := loadGroup(groupPath, group, since, until)
		if err != nil {
			fmt.Printf("failed to parse group %s: %v\n", group, err)
		}
		rule := cfg.GetRule(group)
		res := detector.AnalyzeGroup(events, rule)
		res.Stats.InvalidLines = invalid
		if res.Group == "" {
			res.Group = group
		}
		groupResults = append(groupResults, res)
		allFindings = append(allFindings, res.Findings...)
	}

	sort.Slice(groupResults, func(i, j int) bool { return groupResults[i].Group < groupResults[j].Group })
	return model.AnalysisResult{Groups: groupResults, Findings: allFindings}, nil
}

func loadGroup(path, group string, since, until *time.Time) ([]model.Event, int, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, 0, err
	}
	var events []model.Event
	invalid := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		filePath := filepath.Join(path, e.Name())
		evs, inv, err := parser.ParseFile(filePath, group, since, until)
		if err != nil {
			fmt.Printf("warn: failed to parse file %s: %v\n", filePath, err)
		}
		events = append(events, evs...)
		invalid += inv
	}
	sort.Slice(events, func(i, j int) bool { return events[i].Timestamp.Before(events[j].Timestamp) })
	return events, invalid, nil
}
