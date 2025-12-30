package rules

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

// Rule defines thresholds per group.
type Rule struct {
	MaxWait            time.Duration
	ExcessRcvRatio     float64
	RcvFloodThreshold  int
	DuplicateRcvRepeat int
}

// Config holds default and overrides.
type Config struct {
	Default   Rule
	Overrides map[string]Rule
}

// Load loads rules from a minimal YAML-like file.
func Load(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, err
	}
	defer f.Close()

	cfg := Config{Overrides: make(map[string]Rule)}
	scanner := bufio.NewScanner(f)
	section := ""
	currentGroup := ""
	tempRules := make(map[string]Rule)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasSuffix(line, ":") {
			key := strings.TrimSuffix(line, ":")
			if key == "default" || key == "overrides" {
				section = key
				currentGroup = ""
				continue
			}
			if section == "overrides" {
				currentGroup = key
				tempRules[currentGroup] = Rule{}
				continue
			}
		}

		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			k := strings.TrimSpace(parts[0])
			v := strings.Trim(strings.TrimSpace(parts[1]), "\" ")
			switch section {
			case "default":
				applyRuleField(&cfg.Default, k, v)
			case "overrides":
				r := tempRules[currentGroup]
				applyRuleField(&r, k, v)
				tempRules[currentGroup] = r
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return Config{}, err
	}

	for k, v := range tempRules {
		cfg.Overrides[k] = v
	}

	// Apply fallbacks for default if missing
	cfg.Default = normalizeRule(cfg.Default)
	for k, r := range cfg.Overrides {
		cfg.Overrides[k] = normalizeRule(r)
	}

	return cfg, nil
}

// GetRule returns rule for group falling back to default.
func (c Config) GetRule(group string) Rule {
	if r, ok := c.Overrides[group]; ok {
		return r
	}
	return c.Default
}

func applyRuleField(r *Rule, key, value string) {
	switch key {
	case "MaxWait":
		if d, err := time.ParseDuration(value); err == nil {
			r.MaxWait = d
		}
	case "ExcessRcvRatio":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			r.ExcessRcvRatio = f
		}
	case "RcvFloodThreshold":
		if n, err := strconv.Atoi(value); err == nil {
			r.RcvFloodThreshold = n
		}
	case "DuplicateRcvRepeat":
		if n, err := strconv.Atoi(value); err == nil {
			r.DuplicateRcvRepeat = n
		}
	}
}

func normalizeRule(r Rule) Rule {
	if r.MaxWait == 0 {
		r.MaxWait = 5 * time.Second
	}
	if r.ExcessRcvRatio == 0 {
		r.ExcessRcvRatio = 1.5
	}
	if r.RcvFloodThreshold == 0 {
		r.RcvFloodThreshold = 3
	}
	if r.DuplicateRcvRepeat == 0 {
		r.DuplicateRcvRepeat = 3
	}
	return r
}
