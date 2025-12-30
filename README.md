# Log Analyzer for Field Devices

## Purpose
The `loganalyzer` CLI scans device logs (serial/gateway/servers) to detect when a request has no response, responses are excessive, or receive floods/duplicates happen. It reads the folder layout such as `ALL`, `GATE1`, `GATE2`, `PING`, `PUMP1`, `SERVER`, `TEMP1`, `WLS1`, `WLS2`, `WLS3`, etc., and reports anomalies per group.

## Installation
```bash
make build
```
This builds `cmd/loganalyzer` into the current module cache. To install globally:
```bash
go install ./cmd/loganalyzer
```

## Usage
```bash
./loganalyzer --root ~/Downloads/underware202408-main/log --json --out report.json
```

Flags:
- `--root` (required): path to the log root folder
- `--config`: YAML rule file path (default `configs/rules.yaml`)
- `--json`: print findings as JSON
- `--out`: save results to a file (JSON)
- `--summary-only`: print anomaly counts per group in text (no detailed findings)
- `--since` / `--until`: analyze only within the time window

## Rule Configuration (configs/rules.yaml)
```yaml
default:
  MaxWait: "5s"
  ExcessRcvRatio: 1.5
  RcvFloodThreshold: 3
  DuplicateRcvRepeat: 3
overrides:
  WLS1:
    MaxWait: "3s"
  PUMP1:
    MaxWait: "3s"
  GATE1:
    MaxWait: "2s"
```
You can add more group overrides as needed.

## Output Example
Console summary example:
```
[WLS1] lines=6 invalid=0 snd=2 rcv=4 missing=1 flood=1 dup=1 excess=true
Findings:
2025-12-30 14:10:10.000 [WLS1] MISSING_RESPONSE - snd without timely rcv
2025-12-30 14:10:14.000 [WLS1] RCV_FLOOD - rcv flood without snd
2025-12-30 14:10:14.000 [WLS1] DUPLICATE_RCV - duplicate rcv payload
2025-12-30 14:10:14.200 [WLS1] EXCESS_RCV - rcv count exceeds ratio
```
JSON output is also supported via `--json` or `--out`.

### How detection works
- **응답없음 (missing response)**: a `snd` event is sent, but no matching `rcv` arrives within the configured wait window.
- **센서고장 (sensor fault)**: a `snd` is followed by an `rcv` that contains an invalid payload such as `(00)` for WLS sensors.

## Summary-only mode (text)
To print only anomaly counts per group (no detailed lines):
```bash
./loganalyzer --root ~/Downloads/underware202408-main/log --summary-only
```

Output format:
```
[WLS1]
- 응답없음: 3
- 센서고장: 5

[GATE1]
- 응답없음: 0
- 센서고장: 7
```

## Extensibility
- Core parsing and detection logic lives under `internal/parser`, `internal/detector`, `internal/rules`, and `internal/report` making it easy to reuse.
- A future Fyne UI can import these packages and call `analyzeRoot` to visualize results.
- To add a new equipment group, drop its folder under the log root and, if needed, add an override rule in `configs/rules.yaml`.

## Development
- Run tests: `make test`
- Format code: `make lint`

Sample logs for tests are in `testdata/log/*`.
