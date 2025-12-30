package parser

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"field-data-monitoring/internal/model"
)

var lineRegex = regexp.MustCompile(`^(?P<ts>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3})\s+(?P<dir>snd|rcv):\s+(?P<payload>.*)$`)

// ParseLine parses a single log line into an Event.
func ParseLine(line, file, group string, lineNo int) (model.Event, error) {
	matches := lineRegex.FindStringSubmatch(line)
	if matches == nil {
		return model.Event{}, fmt.Errorf("invalid format")
	}
	tsStr := matches[1]
	dir := matches[2]
	payload := matches[3]

	ts, err := time.Parse("2006-01-02 15:04:05.000", tsStr)
	if err != nil {
		return model.Event{}, err
	}

	event := model.Event{
		Timestamp:  ts,
		Group:      group,
		Dir:        dir,
		PayloadRaw: strings.TrimSpace(payload),
		File:       file,
		Line:       lineNo,
	}

	if strings.HasPrefix(strings.TrimSpace(payload), "(") && strings.HasSuffix(strings.TrimSpace(payload), ")") {
		bytes, err := parseBytes(payload)
		if err == nil {
			event.PayloadBytes = bytes
		}
	} else if strings.Contains(payload, "=") {
		event.KV = parseKV(payload)
	}

	return event, nil
}

// parseBytes parses payload like (FA, FF)
func parseBytes(payload string) ([]byte, error) {
	payload = strings.TrimSpace(payload)
	payload = strings.TrimPrefix(payload, "(")
	payload = strings.TrimSuffix(payload, ")")
	parts := strings.Split(payload, ",")
	result := make([]byte, 0, len(parts))
	for _, p := range parts {
		val := strings.TrimSpace(p)
		if val == "" {
			continue
		}
		var b byte
		if strings.HasPrefix(strings.ToLower(val), "0x") {
			v, err := strconv.ParseUint(val[2:], 16, 8)
			if err != nil {
				return nil, err
			}
			b = byte(v)
		} else if len(val) == 2 && isHex(val) {
			v, err := strconv.ParseUint(val, 16, 8)
			if err != nil {
				return nil, err
			}
			b = byte(v)
		} else {
			v, err := strconv.ParseUint(val, 10, 8)
			if err != nil {
				return nil, err
			}
			b = byte(v)
		}
		result = append(result, b)
	}
	return result, nil
}

func isHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// parseKV parses key=value pairs separated by comma.
func parseKV(payload string) map[string]string {
	kv := make(map[string]string)
	parts := strings.Split(payload, ",")
	for _, part := range parts {
		pair := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(pair) == 2 {
			kv[pair[0]] = pair[1]
		}
	}
	return kv
}

// ParseFile streams a log file and sends events via channel. Invalid lines increment invalid counter.
func ParseFile(path, group string, since, until *time.Time) ([]model.Event, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	var events []model.Event
	invalid := 0
	scanner := bufio.NewScanner(f)
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		ev, err := ParseLine(line, path, group, lineNo)
		if err != nil {
			invalid++
			continue
		}
		if since != nil && ev.Timestamp.Before(*since) {
			continue
		}
		if until != nil && ev.Timestamp.After(*until) {
			continue
		}
		events = append(events, ev)
	}
	if err := scanner.Err(); err != nil {
		return events, invalid, errors.New("scan error: " + err.Error())
	}
	return events, invalid, nil
}
