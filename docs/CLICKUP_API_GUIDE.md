# ClickUp API로 Task(이슈) 등록하는 방법

**개발 가이드 문서**

- **작성일**: 2026년 1월 1일
- **버전**: 1.0
- **목적**: ClickUp API를 사용하여 프로그래밍 방식으로 Task(이슈)를 등록하는 방법에 대한 개발 가이드

---

## 목차

1. [API 엔드포인트](#1-api-엔드포인트)
2. [필수 요구사항](#2-필수-요구사항)
3. [필수 및 선택 파라미터](#3-필수-및-선택-파라미터)
4. [요청 예시](#4-요청-예시)
5. [응답 예시](#5-응답-예시)
6. [Custom Fields 설정](#6-custom-fields-설정)
7. [권장 사항](#7-권장-사항)
8. [관련 API 및 참고자료](#8-관련-api-및-참고자료)

---

## 1. API 엔드포인트

**HTTP Method**: `POST`

**URL**: 
```
https://api.clickup.com/api/v2/list/{list_id}/task
```

**공식 문서**: https://developer.clickup.com/reference/createtask

---

## 2. 필수 요구사항

### 2.1 인증

- **Authorization Header**: `Authorization: Bearer {token}`
- token은 다음 중 하나:
  - ClickUp OAuth 토큰 (권장, 더 안전함)
  - Personal API Token (ClickUp 계정 설정에서 생성)
- **OAuth 설정 가이드**: https://developer.clickup.com/docs/authentication

### 2.2 List ID 확보

- Task를 생성할 List의 ID 필요
- ClickUp UI에서 확인하거나 Get Lists API로 조회
- **Get Lists API**: https://developer.clickup.com/reference/getlists
- **List ID 조회 예시**:
  ```
  GET https://api.clickup.com/api/v2/team/{team_id}/list?archived=false
  Authorization: Bearer {token}
  ```

### 2.3 헤더 설정

```
Content-Type: application/json
Authorization: Bearer {your_api_token}
```

---

## 3. 필수 및 선택 파라미터

### 3.1 필수 파라미터 (Required)

| 파라미터 | 타입 | 설명 |
|---------|------|------|
| `name` | string | Task의 제목 (필수) |
| `list_id` | number | Task를 생성할 List ID (경로 파라미터, 필수) |

### 3.2 선택 파라미터 (Optional)

#### 기본 정보
- `description` (string): Task의 설명
- `markdown_content` (string): Markdown 형식의 설명 
  - description과 markdown_content 모두 제공되면 markdown_content 사용

#### 담당자 및 그룹
- `assignees` (array of integers): 담당자 User ID 배열
  - 예: `[123, 456, 789]`
- `group_assignees` (array of strings): 그룹 담당자 배열
  - 예: `["group_1", "group_2"]`

#### 분류 및 상태
- `tags` (array of strings): Task 태그 배열
  - 예: `["urgent", "backend", "api"]`
- `status` (string): Task 상태
  - 예: `"to do"`, `"in progress"`, `"done"` 등
- `priority` (integer): 우선순위 (1~4)
  - `1` = Urgent (긴급)
  - `2` = High (높음)
  - `3` = Normal (보통)
  - `4` = Low (낮음)

#### 날짜 및 시간
- `due_date` (integer): 마감일 (Unix timestamp, 밀리초)
  - 예: `1704067200000` (2024-01-01)
- `due_date_time` (boolean): 마감일에 시간 포함 여부
  - `true` = 시간 포함, `false` = 날짜만 (기본값)
- `start_date` (integer): 시작일 (Unix timestamp, 밀리초)
- `start_date_time` (boolean): 시작일에 시간 포함 여부

#### 예상 작업량
- `time_estimate` (integer): 예상 소요시간 (분 단위)
  - 예: `3600` (60분 = 1시간)
- `points` (number): Sprint Points
  - 예: `5`, `8`, `13`

#### 기타
- `notify_all` (boolean): 모든 사용자에게 알림 발송
  - `true` = 알림 발송, `false` = 기본값
- `parent` (string): 부모 Task ID (서브태스크용)
  - 예: `"8xdfdjbgd"`
- `links_to` (string): 연결할 Task ID (작업 의존성)
  - 예: `"8xdfm9vmz"`
- `check_required_custom_fields` (boolean): Custom Field 검증
  - `true` = 필수 Custom Field 검증, `false` = 기본값 (검증 안함)
- `custom_fields` (array): Custom Fields (아래 6번 섹션 참고)
- `custom_item_id` (number): Custom Task Type ID
  - `null` = 표준 Task (기본값)
  - 참고: https://developer.clickup.com/reference/getcustomitems

---

## 4. 요청 예시

### 4.1 기본 예시 (Python)

```python
import requests
import json

# 설정
api_token = "your_api_token"
list_id = "901234567890"  # 작업을 생성할 List ID

# 헤더
headers = {
    "Authorization": f"Bearer {api_token}",
    "Content-Type": "application/json"
}

# 요청 본문
data = {
    "name": "API를 통한 새 이슈",
    "description": "이것은 API로 생성된 이슈입니다.",
    "priority": 2,  # 높음
    "assignees": [123],  # 담당자 ID
    "tags": ["api", "urgent"],
    "due_date": 1704067200000  # 2024-01-01
}

# API 요청
url = f"https://api.clickup.com/api/v2/list/{list_id}/task"
response = requests.post(url, headers=headers, json=data)

# 결과
if response.status_code == 200:
    task = response.json()
    print(f"✅ 작업 생성 성공!")
    print(f"Task ID: {task['id']}")
    print(f"Task Name: {task['name']}")
else:
    print(f"❌ 오류: {response.status_code}")
    print(response.json())
```

### 4.2 고급 예시 (모든 옵션 포함)

```python
import requests
from datetime import datetime, timedelta

api_token = "your_api_token"
list_id = "901234567890"

headers = {
    "Authorization": f"Bearer {api_token}",
    "Content-Type": "application/json"
}

# 마감일을 오늘로부터 일주일 후로 설정 (Unix timestamp)
due_date_timestamp = int((datetime.now() + timedelta(days=7)).timestamp() * 1000)

data = {
    "name": "버그 수정 - 로그인 페이지",
    "markdown_content": """# 문제 설명
로그인 페이지에서 비밀번호 초기화 기능이 작동하지 않습니다.

## 재현 방법
1. 로그인 페이지 접속
2. "비밀번호 초기화" 클릭
3. 오류 메시지 확인

## 예상 결과
이메일 입력 화면으로 이동해야 함""",
    "assignees": [123, 456],  # 여러 담당자
    "priority": 1,  # 긴급
    "status": "to do",
    "tags": ["bug", "frontend", "high-priority"],
    "due_date": due_date_timestamp,
    "due_date_time": True,  # 시간 포함
    "start_date": int(datetime.now().timestamp() * 1000),
    "time_estimate": 7200,  # 120분 = 2시간
    "points": 8,  # Sprint Points
    "notify_all": True,
    "custom_fields": [
        {
            "id": "custom_field_id_1",
            "value": "Production"  # Severity Level
        }
    ]
}

url = f"https://api.clickup.com/api/v2/list/{list_id}/task"
response = requests.post(url, headers=headers, json=data)

if response.status_code == 200:
    print("✅ Task 생성 성공!")
else:
    print(f"❌ 오류: {response.status_code}")
    print(response.json())
```

### 4.3 cURL 예시

```bash
curl -X POST "https://api.clickup.com/api/v2/list/901234567890/task" \
  -H "Authorization: Bearer your_api_token" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "API 이슈 테스트",
    "description": "이것은 API로 생성된 이슈입니다.",
    "priority": 2,
    "tags": ["api"],
    "due_date": 1704067200000
  }'
```

### 4.4 JavaScript/Node.js 예시

```javascript
const axios = require('axios');

const apiToken = 'your_api_token';
const listId = '901234567890';

const headers = {
    'Authorization': `Bearer ${apiToken}`,
    'Content-Type': 'application/json'
};

const data = {
    name: 'API를 통한 새 이슈',
    description: '이것은 API로 생성된 이슈입니다.',
    priority: 2,
    assignees: [123],
    tags: ['api', 'urgent'],
    due_date: 1704067200000
};

axios.post(`https://api.clickup.com/api/v2/list/${listId}/task`, data, { headers })
    .then(response => {
        console.log('✅ 작업 생성 성공!');
        console.log('Task ID:', response.data.id);
        console.log('Task Name:', response.data.name);
    })
    .catch(error => {
        console.error('❌ 오류:', error.response.status);
        console.error(error.response.data);
    });
```

---

## 5. 응답 예시

### 성공적인 Task 생성 후 응답 (HTTP 200)

```json
{
  "id": "8xdfdjbgd",
  "custom_id": "CUS-001",
  "name": "API를 통한 새 이슈",
  "text_content": "이것은 API로 생성된 이슈입니다.",
  "description": "이것은 API로 생성된 이슈입니다.",
  "status": {
    "status": "to do",
    "color": "#87CEEB",
    "orderindex": 0,
    "type": "open"
  },
  "creator": {
    "id": 12345,
    "username": "api_user",
    "color": "#FF5733",
    "profilePicture": "https://..."
  },
  "assignees": [
    {
      "id": 123,
      "username": "john_doe",
      "color": "#1E90FF",
      "profilePicture": "https://..."
    }
  ],
  "priority": {
    "id": "2",
    "priority": "high",
    "color": "#FFAA00",
    "orderindex": "1"
  },
  "due_date": 1704067200000,
  "start_date": 1703462400000,
  "time_estimate": 7200,
  "points": 8,
  "tags": ["api", "urgent"],
  "date_created": "1704067200000",
  "date_updated": "1704067200000",
  "url": "https://app.clickup.com/t/8xdfdjbgd",
  "list": {
    "id": "901234567890"
  },
  "folder": {
    "id": "12345"
  },
  "space": {
    "id": "123"
  }
}
```

### 응답 주요 필드 설명

| 필드 | 설명 |
|------|------|
| `id` | Task의 고유 ID (이후 Task 업데이트/조회 시 사용) |
| `custom_id` | ClickUp에서 자동 생성한 Task 번호 (예: CUS-001) |
| `name` | Task 제목 |
| `status` | Task 상태 정보 |
| `priority` | 우선순위 정보 |
| `assignees` | 담당자 정보 배열 |
| `url` | ClickUp에서 Task에 접근할 수 있는 URL |
| `date_created` | Task 생성 시간 (Unix timestamp) |
| `date_updated` | Task 마지막 수정 시간 (Unix timestamp) |

---

## 6. Custom Fields 설정

Custom Field가 있는 경우 create task 시 설정할 수 있습니다.

### 6.1 Custom Fields 조회 방법

먼저 Workspace에서 사용 가능한 Custom Field를 조회해야 합니다:

**API**: https://developer.clickup.com/reference/getaccessiblecustomfields

**Endpoint**:
```
GET https://api.clickup.com/api/v2/team/{team_id}/custom_field
```

**응답 예시**:
```json
{
  "custom_fields": [
    {
      "id": "field_id_1",
      "name": "Severity",
      "type": "single_select",
      "type_config": {
        "options": [
          { "id": "opt_1", "name": "Critical", "color": "red" },
          { "id": "opt_2", "name": "High", "color": "orange" },
          { "id": "opt_3", "name": "Medium", "color": "yellow" }
        ]
      }
    },
    {
      "id": "field_id_2",
      "name": "Release Version",
      "type": "text"
    }
  ]
}
```

### 6.2 Create Task 시 Custom Fields 설정

```json
{
  "name": "버그 수정",
  "description": "로그인 페이지 버그",
  "custom_fields": [
    {
      "id": "field_id_1",
      "value": "opt_1"
    },
    {
      "id": "field_id_2",
      "value": "v2.5.0"
    }
  ]
}
```

### 6.3 주의사항

Custom Field 타입에 따라 value 형식이 다릅니다:

| 타입 | 예시 |
|------|------|
| `text` | `"some text"` |
| `number` | `123` |
| `email` | `"user@example.com"` |
| `url` | `"https://example.com"` |
| `phone` | `"+1234567890"` |
| `date` | `"2024-01-01"` |
| `single_select` | `"opt_1"` |
| `multiple_select` | `["opt_1", "opt_2"]` |
| `user` | `123` |
| `currency` | `{"value": 100, "currency": "USD"}` |

**주의**:
- 필수 Custom Field가 있으면 `check_required_custom_fields: true`를 설정하여 검증 활성화
- Custom Field를 비우려면 `"value": null` 사용

---

## 7. 권장 사항

### 7.1 API 사용 전 확인사항

#### 1. List ID 확보
- "Get Lists" API로 먼저 조회하여 정확한 List ID 확보
- **API**: https://developer.clickup.com/reference/getlists

#### 2. User ID 확인
- "Get Workspace Members" API로 담당자의 정확한 User ID 확보
- **API**: https://developer.clickup.com/reference/gettaskmembers
- **Endpoint**: 
  ```
  GET https://api.clickup.com/api/v2/team/{team_id}/member
  ```

#### 3. Custom Fields 검증
- "Get Custom Fields" API로 사용 가능한 Custom Field 목록 조회
- **API**: https://developer.clickup.com/reference/getaccessiblecustomfields
- Custom Field ID와 option ID 정확성 확인

#### 4. 인증 토큰 검증
- 토큰이 유효한지 "Get Authorized User" API로 확인
- **API**: https://developer.clickup.com/reference/getauthorizeduser

### 7.2 요청 작성 시 Best Practice

#### 1. 에러 처리

| HTTP 상태 | 의미 | 대처 방법 |
|-----------|------|---------|
| 400 | Bad Request | 요청 본문 검토 |
| 401 | Unauthorized | 토큰 검증 |
| 403 | Forbidden | 사용자 권한 확인 |
| 429 | Rate Limit | 재시도 (exponential backoff) |

자세한 Rate Limit: https://developer.clickup.com/docs/rate-limits

#### 2. 타임스탐프 형식
- Unix timestamp는 반드시 밀리초 단위 사용
- 예: `1704067200000` (1704067200 * 1000)
- JavaScript: `Date.now()` 사용
- Python: `int(datetime.now().timestamp() * 1000)`

#### 3. Markdown 형식 사용
- `description`과 `markdown_content` 중 하나만 선택
- `markdown_content`가 있으면 `description`은 무시됨
- 지원: 제목, 강조, 리스트, 링크, 코드 블록, 인용구
- 자세한 가이드: https://developer.clickup.com/docs/tasks

#### 4. 대량 작업 처리
- 여러 Task를 생성할 때는 "Bulk Create Task" API 사용 고려
- **API**: https://developer.clickup.com/reference/createbulktasks
- 한 번에 최대 100개 Task 생성 가능

#### 5. 응답 활용
- 생성된 Task의 ID를 반드시 저장
- 나중에 Task 업데이트나 조회 시 필요
- URL을 이용하여 ClickUp에서 즉시 Task 확인 가능

### 7.3 보안 관련 권장사항

#### 1. API 토큰 관리
- 토큰을 코드에 하드코딩하지 말 것
- 환경 변수나 설정 파일로 관리 (예: .env)
- 정기적으로 토큰 갱신

#### 2. OAuth 사용
- 가능하면 Personal API Token 대신 OAuth 사용
- OAuth: https://developer.clickup.com/docs/authentication

#### 3. HTTPS 사용
- API 통신은 반드시 HTTPS 사용
- HTTP는 사용하지 말 것

### 7.4 로깅 및 모니터링

```python
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

try:
    response = requests.post(url, headers=headers, json=data)
    response.raise_for_status()
    logger.info(f"Task created successfully: {response.json()['id']}")
except requests.exceptions.RequestException as e:
    logger.error(f"API request failed: {e}")
    logger.error(f"Response: {e.response.text if hasattr(e, 'response') else 'N/A'}")
```

---

## 8. 관련 API 및 참고자료

### 8.1 자주 사용하는 관련 API

#### Task 관리
- **Create Task** (이 문서에서 설명): https://developer.clickup.com/reference/createtask
- **Get Task** (Task 조회): https://developer.clickup.com/reference/gettask
- **Update Task** (Task 수정): https://developer.clickup.com/reference/updatetask
- **Delete Task** (Task 삭제): https://developer.clickup.com/reference/deletetask
- **Get Workspace Tasks** (모든 Task 조회): https://developer.clickup.com/reference/gettasks
- **Create Bulk Tasks** (대량 Task 생성): https://developer.clickup.com/reference/createbulktasks

#### List 및 폴더
- **Get Lists** (List 조회): https://developer.clickup.com/reference/getlists
- **Create List**: https://developer.clickup.com/reference/createlist
- **Get Folders**: https://developer.clickup.com/reference/getfolders
- **Create Folder**: https://developer.clickup.com/reference/createfolder

#### 담당자 및 멤버
- **Get Workspace Members** (멤버 조회): https://developer.clickup.com/reference/gettaskmembers
- **Find Member by Name**: https://developer.clickup.com/reference/findmember

#### Custom Fields
- **Get Custom Fields** (Custom Field 조회): https://developer.clickup.com/reference/getaccessiblecustomfields
- **Set Custom Field Value** (Custom Field 값 설정): https://developer.clickup.com/reference/setcustomfieldvalue

#### Comments
- **Create Task Comment** (댓글 추가): https://developer.clickup.com/reference/commentontask
- **Get Task Comments** (댓글 조회): https://developer.clickup.com/reference/gettaskcomments

#### Time Tracking
- **Start Time Tracking**: https://developer.clickup.com/reference/starttimetracking
- **Stop Time Tracking**: https://developer.clickup.com/reference/stoptimetracking
- **Add Time Entry**: https://developer.clickup.com/reference/addtimeentry

### 8.2 중요 공식 문서

#### ClickUp API 공식 문서
- **API 메인 페이지**: https://developer.clickup.com/
- **API 가이드 문서**: https://developer.clickup.com/docs
- **API 레퍼런스**: https://developer.clickup.com/reference
- **인증 가이드**: https://developer.clickup.com/docs/authentication
- **Rate Limit 정보**: https://developer.clickup.com/docs/rate-limits
- **Task 가이드**: https://developer.clickup.com/docs/tasks

#### ClickUp 공식 페이지
- **ClickUp 메인 페이지**: https://clickup.com
- **ClickUp 헬프 센터**: https://help.clickup.com/

### 8.3 유용한 도구

#### API 테스트 도구
- **Postman**: https://www.postman.com/
  - ClickUp API Collection을 import하여 각 엔드포인트 테스트 가능
  
- **cURL**: 터미널에서 직접 API 호출
  
- **Insomnia**: https://insomnia.rest/
  - Postman과 유사한 기능

#### JSON 검증 도구
- **JSONLint**: https://jsonlint.com/
- **Online JSON Validator**: https://www.jsonschemavalidator.net/

#### Unix Timestamp 변환 도구
- **Epoch Converter**: https://www.epochconverter.com/
- **Unix Timestamp Converter**: https://www.unixtimestamp.com/

### 8.4 문제 해결 (Troubleshooting)

| 문제 | 해결 방법 |
|------|---------|
| "Invalid List ID" 오류 | Get Lists API로 정확한 List ID 확인 |
| "Invalid assignee" 오류 | Get Workspace Members API로 정확한 User ID 확인 |
| "Rate limit exceeded" (HTTP 429) | 요청을 줄이거나 exponential backoff 사용 ([Rate Limit](https://developer.clickup.com/docs/rate-limits)) |
| "Invalid custom field ID" 오류 | Get Custom Fields API로 정확한 Custom Field ID 확인 |
| "Required field missing" 오류 | 요청 본문에 필수 파라미터(name) 포함 확인 |
| 401 Unauthorized | Authorization 헤더의 토큰 확인, Get Authorized User API로 토큰 유효성 검증 |

### 8.5 추가 학습 자료

- **ClickUp API 개발자 포럼**: https://developers.clickup.com
- **ClickUp YouTube 채널**: https://www.youtube.com/c/ClickUp
- **ClickUp Community**: https://community.clickup.com/

---

## 문서 정보

- **작성일**: 2026년 1월 1일
- **버전**: 1.0
- **마지막 업데이트**: 2026년 1월 1일
- **저자**: Development Team
- **라이선스**: Internal Use Only

이 문서는 ClickUp API를 사용하여 Task를 등록하는 개발자들을 위한 참고 자료입니다.
정보는 ClickUp 공식 API 문서를 기반으로 작성되었습니다.