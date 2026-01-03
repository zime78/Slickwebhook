# 개발 지침 (Development Guidelines)

## 📝 언어 정책

> **모든 문서와 코드 내용은 한국어로 작성합니다.**

### 적용 범위

| 항목 | 언어 | 비고 |
| ---- | ---- | ---- |
| README, 문서 | 🇰🇷 한국어 | 필수 |
| 코드 주석 | 🇰🇷 한국어 | 필수 |
| 커밋 메시지 | 🇰🇷 한국어 | 권장 |
| 변수/함수명 | 🇺🇸 영어 | 코드 호환성 |
| API 응답 메시지 | 🇰🇷 한국어 | 권장 |
| 로그 메시지 | 🇰🇷 한국어 | 권장 |

---

## 📁 프로젝트 구조

```text
SlickWebhook/
├── README.md                   # 프로젝트 소개 및 사용법
├── CONTRIBUTING.md             # 개발 지침 (이 파일)
├── payload.json                # Agent Hook 페이로드
├── scripts/
│   └── send_hook.sh            # Agent Hook 전송 스크립트
└── postman/
    ├── clickup_agent_hook_collection.json   # Postman Collection
    └── clickup_agent_environment.json       # Postman Environment
```

---

## ✍️ 문서 작성 규칙

1. **제목과 설명은 한국어로 작성**
2. **기술 용어는 영문 병기 허용** (예: "웹훅(Webhook)")
3. **코드 블록 내 주석도 한국어**
4. **이모지 사용 권장** - 가독성 향상

### 예시

```bash
# ✅ 좋은 예
# API 토큰을 환경변수에서 가져옵니다
CLICKUP_API_TOKEN="${CLICKUP_API_TOKEN:-기본값}"

# ❌ 나쁜 예
# Get API token from environment variable
CLICKUP_API_TOKEN="${CLICKUP_API_TOKEN:-default}"
```

---

## 🔄 커밋 메시지 규칙

```text
<타입>: <설명>

[본문 (선택)]
```

### 타입

| 타입 | 설명 |
| ---- | ---- |
| `기능` | 새로운 기능 추가 |
| `수정` | 버그 수정 |
| `문서` | 문서 변경 |
| `리팩터` | 코드 리팩토링 |
| `설정` | 설정 파일 변경 |

### 예시

```text
기능: Slack → ClickUp Agent Hook 스크립트 추가

- send_hook.sh 생성
- payload.json 설정 파일 추가
- Postman Collection/Environment 추가
```

---

## 🛠️ 환경 설정

### 필수 환경변수

| 변수명 | 설명 | 필수 |
| ------ | ---- | ---- |
| `CLICKUP_API_TOKEN` | ClickUp API 토큰 | ✅ |

### 권장 도구

- `jq` - JSON 포맷팅 (`brew install jq`)
- `curl` - API 호출
- Postman - API 테스트

---

## 📋 체크리스트

새 기능 추가 시:

- [ ] 모든 문서가 한국어로 작성되었는가?
- [ ] 코드 주석이 한국어로 작성되었는가?
- [ ] README가 업데이트되었는가?
- [ ] 실행 테스트를 완료했는가?
