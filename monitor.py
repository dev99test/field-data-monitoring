"""
Field data monitoring and log analysis utility.

This script scans device log directories, extracts up to the most recent
30 response values per sensor, and generates an analysis log file. The
resulting log can be sent to a collector server or stored locally.
"""
from __future__ import annotations

import argparse
import configparser
import dataclasses
from datetime import datetime
import json
from pathlib import Path
import re
import sys
import urllib.error
import urllib.request
from typing import Dict, Iterable, List, Tuple


ACTION_MAP: Dict[str, str] = {
    "road_level": "수위계 교체 / 케이블 교체",
    "sump_level": "수위계 교체 / 케이블 교체",
    "pump": "펌프로거보드 교체",
    "breaker": "차단기업체 확인(증상전달)",
    "thermo": "교체",
}

CATEGORY_LABELS: Dict[str, str] = {
    "road_level": "도로수위계",
    "sump_level": "집수정수위계",
    "pump": "펌프로거",
    "breaker": "차단기",
    "thermo": "온도계",
}


@dataclasses.dataclass
class SensorResult:
    name: str
    category: str
    log_file: Path | None
    responses: List[str]
    status: str
    reason: str
    action: str | None

    def to_dict(self) -> Dict[str, object]:
        return {
            "name": self.name,
            "category": self.category,
            "log_file": str(self.log_file) if self.log_file else None,
            "responses": self.responses,
            "status": self.status,
            "reason": self.reason,
            "action": self.action,
        }


@dataclasses.dataclass
class AnalyzerConfig:
    pc_ip: str
    log_root: Path
    output_dir: Path
    collector_ip: str
    collector_port: int
    send_enabled: bool
    sensors: Dict[str, List[str]]


HEX_PAIR_PATTERN = re.compile(r"\b([0-9A-Fa-f]{2})\b")


def load_config(path: Path) -> AnalyzerConfig:
    parser = configparser.ConfigParser()
    if not path.exists():
        raise FileNotFoundError(f"Config file not found: {path}")
    parser.read(path)

    general = parser["general"]
    server = parser["server"]
    sensors_section = parser["sensors"]

    sensors: Dict[str, List[str]] = {}
    for key, value in sensors_section.items():
        # Normalize keys like "road_level" and trim whitespace around names
        names = [name.strip() for name in value.split(",") if name.strip()]
        sensors[key] = names

    return AnalyzerConfig(
        pc_ip=general.get("pc_ip", "0.0.0.0"),
        log_root=Path(general.get("log_root", "log")),
        output_dir=Path(general.get("output_dir", "analysis_logs")),
        collector_ip=server.get("collector_ip", "127.0.0.1"),
        collector_port=server.getint("collector_port", 8000),
        send_enabled=server.getboolean("send_enabled", False),
        sensors=sensors,
    )


def latest_file_in(directory: Path) -> Path | None:
    if not directory.exists() or not directory.is_dir():
        return None
    files = [path for path in directory.iterdir() if path.is_file()]
    if not files:
        return None
    return max(files, key=lambda p: p.stat().st_mtime)


def extract_responses(file_path: Path, limit: int = 30) -> List[str]:
    text = file_path.read_text(encoding="utf-8", errors="ignore")
    all_values = [match.upper() for match in HEX_PAIR_PATTERN.findall(text)]
    if len(all_values) > limit:
        return all_values[-limit:]
    return all_values


def analyze_sensor(log_root: Path, sensor_name: str, category_key: str) -> SensorResult:
    category_label = CATEGORY_LABELS.get(category_key, category_key)
    target_dir = log_root / sensor_name
    latest = latest_file_in(target_dir)

    if latest is None:
        return SensorResult(
            name=sensor_name,
            category=category_label,
            log_file=None,
            responses=[],
            status="FAULT",
            reason="로그 파일이 없어 응답을 확인할 수 없음",
            action=ACTION_MAP.get(category_key),
        )

    responses = extract_responses(latest)
    if not responses:
        return SensorResult(
            name=sensor_name,
            category=category_label,
            log_file=latest,
            responses=[],
            status="FAULT",
            reason="응답 없음",
            action=ACTION_MAP.get(category_key),
        )

    action = ACTION_MAP.get(category_key)
    if any(value == "00" for value in responses):
        return SensorResult(
            name=sensor_name,
            category=category_label,
            log_file=latest,
            responses=responses,
            status="FAULT",
            reason="응답 값에 00 포함",
            action=action,
        )

    if len(responses) < 30:
        return SensorResult(
            name=sensor_name,
            category=category_label,
            log_file=latest,
            responses=responses,
            status="FAULT",
            reason="응답이 30개 미만",
            action=action,
        )

    return SensorResult(
        name=sensor_name,
        category=category_label,
        log_file=latest,
        responses=responses,
        status="OK",
        reason="정상",
        action=None,
    )


def render_report(config: AnalyzerConfig, results: Iterable[SensorResult]) -> Tuple[str, str]:
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    filename = f"{config.pc_ip}_{timestamp}.log"
    lines: List[str] = [
        f"분석시간: {timestamp}",
        f"PC IP: {config.pc_ip}",
        f"Collector: {config.collector_ip}:{config.collector_port}",
        f"로그 경로: {config.log_root.resolve()}",
        "",
    ]

    for result in results:
        lines.append(f"센서: {result.name} ({result.category})")
        if result.log_file:
            lines.append(f"  로그 파일: {result.log_file}")
        else:
            lines.append("  로그 파일: 없음")
        lines.append(f"  상태: {result.status} ({result.reason})")
        if result.responses:
            lines.append(f"  추출 응답값(최대 30): {' '.join(result.responses)}")
        else:
            lines.append("  추출 응답값(최대 30): 없음")
        if result.action:
            lines.append(f"  조치방법: {result.action}")
        lines.append("")

    return filename, "\n".join(lines).strip() + "\n"


def send_log_file(target_ip: str, target_port: int, log_path: Path) -> bool:
    url = f"http://{target_ip}:{target_port}/logs"
    data = log_path.read_bytes()
    request = urllib.request.Request(url, data=data, method="POST")
    request.add_header("Content-Type", "text/plain; charset=utf-8")
    try:
        with urllib.request.urlopen(request, timeout=5) as response:
            return 200 <= response.status < 300
    except urllib.error.URLError:
        return False


def analyze(config_path: Path) -> Path:
    config = load_config(config_path)
    results: List[SensorResult] = []

    for category_key, names in config.sensors.items():
        for name in names:
            results.append(analyze_sensor(config.log_root, name, category_key))

    config.output_dir.mkdir(parents=True, exist_ok=True)
    filename, report = render_report(config, results)
    output_path = config.output_dir / filename
    output_path.write_text(report, encoding="utf-8")

    if config.send_enabled:
        success = send_log_file(config.collector_ip, config.collector_port, output_path)
        status_text = "성공" if success else "실패"
        print(f"수집제어서버 전송: {status_text} ({output_path})")
    else:
        print(f"로그 생성 완료: {output_path}")

    # Also emit a JSON summary for integrations
    summary_path = output_path.with_suffix(".json")
    summary = {
        "pc_ip": config.pc_ip,
        "collector": f"{config.collector_ip}:{config.collector_port}",
        "generated_at": datetime.now().isoformat(timespec="seconds"),
        "results": [result.to_dict() for result in results],
    }
    summary_path.write_text(json.dumps(summary, ensure_ascii=False, indent=2), encoding="utf-8")

    return output_path


def parse_args(argv: List[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="현장 제어반 로그 분석기")
    parser.add_argument(
        "--config",
        type=Path,
        default=Path("config.ini"),
        help="설정 파일 경로 (기본값: config.ini)",
    )
    return parser.parse_args(argv)


def main(argv: List[str] | None = None) -> int:
    args = parse_args(argv or sys.argv[1:])
    try:
        analyze(args.config)
    except FileNotFoundError as exc:
        print(exc)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
