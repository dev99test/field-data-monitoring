# field-data-monitoring

현장 제어반의 센서 로그를 수집·분석하고 결과를 수집제어서버로 전달할 수 있는 간단한 유틸리티입니다.

## 구성 파일 (config.ini)
- `general`
  - `pc_ip`: 분석 PC의 IP. 생성되는 로그 파일 이름에 포함됩니다.
  - `log_root`: 센서 로그가 저장된 루트 경로. 기본값은 `log`.
  - `output_dir`: 분석 로그가 저장되는 위치. 기본값은 `analysis_logs`.
- `server`
  - `collector_ip` / `collector_port`: 분석 로그를 전송할 수집제어서버 주소.
  - `send_enabled`: `true`이면 분석 결과를 HTTP POST(`/logs`)로 전송 시도.
- `sensors`: ALL, SERVER, PING을 제외한 센서 이름을 유형별로 쉼표로 나열합니다.
  - `road_level`: 도로수위계 목록
  - `sump_level`: 집수정수위계 목록
  - `pump`: 펌프로거 목록
  - `breaker`: 차단기 목록
  - `thermo`: 온도계 목록

## 사용 방법
1. `config.ini`의 센서 개수와 서버 IP를 환경에 맞게 수정합니다.
2. 로그 루트(`log_root`) 아래에 각 센서 이름과 동일한 폴더(WLS1, GATE1, PUMP1 등)와 로그 파일이 존재해야 합니다.
3. 분석 실행:
   ```bash
   python monitor.py --config config.ini
   ```
4. 생성 결과
   - `output_dir`에 `PC_IP_YYYYMMDD_HHMMSS.log`와 동일한 JSON 요약 파일이 생성됩니다.
   - 응답이 없거나 `00`이 포함되거나 30개 미만일 경우 조치 방법(센서 교체, 케이블 교체 등)이 로그에 기록됩니다.
   - `send_enabled=true`면 수집제어서버(`http://collector_ip:collector_port/logs`)로 로그를 POST 전송합니다.

## 분석 규칙
- 각 센서 로그에서 가장 최근 파일을 골라 최대 30개의 2자리 HEX 응답을 추출합니다.
- 응답 없음, 응답 값 `00`, 응답 개수 30개 미만은 모두 고장으로 간주합니다.
- 조치 방법
  - 수위계(도로/집수정): 수위계 교체, 케이블 교체
  - 펌프: 펌프로거보드 교체
  - 차단기: 차단기업체 확인(증상전달)
  - 온도계: 교체

## 기타
- 모든 동작은 표준 라이브러리만 사용하며 별도 의존성이 없습니다.
- 로그 경로/파일이 없더라도 실행이 가능하며, 해당 센서는 응답 없음으로 표시됩니다.
