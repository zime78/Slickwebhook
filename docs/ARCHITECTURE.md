# SlickWebhook 아키텍처 문서

## 개요

SlickWebhook은 Slack 채널을 실시간으로 모니터링하고, 새 메시지 감지 시 ClickUp 태스크를 자동 생성하는 Go 기반 서비스입니다.

## 시스템 아키텍처

### 전체 구조

```mermaid
flowchart TB
    subgraph External["외부 서비스"]
        SLACK[("Slack API")]
        CLICKUP[("ClickUp API")]
    end

    subgraph SlickWebhook["SlickWebhook 서비스"]
        MAIN["main.go<br/>(엔트리포인트)"]
        CONFIG["config.Loader"]
        MONITOR["monitor.Service"]

        subgraph Handlers["이벤트 핸들러"]
            CHAIN["ChainHandler"]
            LOG["LogHandler"]
            FWD["ForwardHandler"]
        end

        SLACK_CLIENT["slack.Client"]
        CLICKUP_CLIENT["clickup.Client"]
        HISTORY["history.FileStore"]
    end

    subgraph Storage["로컬 저장소"]
        CONFIG_FILE[("config.ini")]
        HISTORY_FILE[("history.json")]
    end

    CONFIG_FILE --> CONFIG
    CONFIG --> MAIN
    MAIN --> MONITOR
    MONITOR --> SLACK_CLIENT
    SLACK_CLIENT <--> SLACK
    MONITOR --> CHAIN
    CHAIN --> LOG
    CHAIN --> FWD
    FWD --> CLICKUP_CLIENT
    CLICKUP_CLIENT <--> CLICKUP
    FWD --> HISTORY
    HISTORY --> HISTORY_FILE
```

### 레이어 구조 (Clean Architecture)

```mermaid
flowchart TB
    subgraph Presentation["Presentation Layer"]
        MAIN["cmd/monitor/main.go"]
    end

    subgraph Application["Application Layer"]
        MONITOR["monitor.Service"]
        HANDLER["handler.EventHandler"]
    end

    subgraph Domain["Domain Layer"]
        MESSAGE["domain.Message"]
        EVENT["domain.Event"]
    end

    subgraph Infrastructure["Infrastructure Layer"]
        SLACK["slack.Client"]
        CLICKUP["clickup.Client"]
        CONFIG["config.Loader"]
        HISTORY["history.Store"]
    end

    MAIN --> MONITOR
    MAIN --> HANDLER
    MONITOR --> EVENT
    HANDLER --> EVENT
    EVENT --> MESSAGE
    MONITOR -.->|interface| SLACK
    HANDLER -.->|interface| CLICKUP
    HANDLER -.->|interface| HISTORY
    MAIN -.->|uses| CONFIG
```

## 컴포넌트 상세

### 1. 도메인 모델 (`internal/domain/`)

핵심 비즈니스 엔티티를 정의합니다.

| 타입 | 설명 |
|------|------|
| `Message` | Slack 메시지 (Timestamp, UserID, BotID, Text, ChannelID, CreatedAt) |
| `Event` | 이벤트 래퍼 (Type, Message, Error, OccurredAt) |
| `EventType` | 이벤트 종류 (`new_message`, `error`) |

### 2. 모니터 서비스 (`internal/monitor/`)

```mermaid
stateDiagram-v2
    [*] --> Idle: Start()
    Idle --> Polling: ticker.C
    Polling --> CheckMessages: GetChannelHistory()
    CheckMessages --> ProcessMessages: 새 메시지 있음
    CheckMessages --> Idle: 새 메시지 없음
    ProcessMessages --> HandleEvent: EventHandler.Handle()
    HandleEvent --> UpdateTimestamp
    UpdateTimestamp --> Idle
    Idle --> [*]: Stop() / ctx.Done()
```

**주요 책임:**

- 폴링 기반 Slack 채널 모니터링
- 마지막 타임스탬프 관리 (중복 방지)
- 이벤트 생성 및 핸들러 위임

### 3. 이벤트 핸들러 (`internal/handler/`)

**Chain of Responsibility 패턴** 적용:

```mermaid
flowchart LR
    EVENT[Event] --> CHAIN[ChainHandler]
    CHAIN --> LOG[LogHandler<br/>로그 출력]
    CHAIN --> FWD[ForwardHandler<br/>ClickUp 전송]
    FWD --> HISTORY[히스토리 저장]
```

| 핸들러 | 역할 |
|--------|------|
| `LogHandler` | 이벤트 로그 출력 |
| `ForwardHandler` | ClickUp 태스크 생성 + 히스토리 관리 |
| `ChainHandler` | 핸들러 체이닝 (순차 실행) |

### 4. 외부 클라이언트

#### Slack Client (`internal/slack/`)

```go
type Client interface {
    GetChannelHistory(ctx context.Context, channelID, oldest string) ([]*domain.Message, error)
}
```

#### ClickUp Client (`internal/clickup/`)

```go
type Client interface {
    CreateTask(ctx context.Context, msg *domain.Message) (*TaskResponse, error)
}
```

### 5. 히스토리 저장소 (`internal/history/`)

```go
type Store interface {
    Add(record *Record)
    Count() int
}
```

- **구현체**: `FileStore` (JSON 파일 기반)
- **제한**: `HISTORY_MAX_SIZE` (기본 100개, FIFO)

## 데이터 흐름

### 메시지 처리 시퀀스

```mermaid
sequenceDiagram
    participant M as Monitor
    participant S as Slack API
    participant H as EventHandler
    participant C as ClickUp API
    participant HS as HistoryStore

    loop 매 PollInterval
        M->>S: GetChannelHistory(oldest)
        S-->>M: messages[]

        alt 새 메시지 있음
            loop 각 메시지
                M->>M: NewMessageEvent(msg)
                M->>H: Handle(event)
                H->>H: LogHandler.Handle()
                H->>C: CreateTask(msg)
                C-->>H: TaskResponse
                H->>HS: Add(record)
            end
            M->>M: lastTimestamp 업데이트
        end
    end
```

## 설정 흐름

```mermaid
flowchart LR
    subgraph 시작시
        A[config.ini] -->|LoadEnvFile| B[환경변수]
        B --> C[os.Getenv]
        C --> D[서비스 초기화]
    end
```

**설정 우선순위**: `config.ini` → 환경변수

## 의존성 그래프

```mermaid
flowchart TD
    MAIN["cmd/monitor/main.go"]

    MAIN --> CONFIG["config"]
    MAIN --> MONITOR["monitor"]
    MAIN --> HANDLER["handler"]
    MAIN --> SLACK["slack"]
    MAIN --> CLICKUP["clickup"]
    MAIN --> HISTORY["history"]

    MONITOR --> DOMAIN["domain"]
    MONITOR --> HANDLER
    MONITOR --> SLACK

    HANDLER --> DOMAIN
    HANDLER --> CLICKUP
    HANDLER --> HISTORY

    SLACK --> DOMAIN
    CLICKUP --> DOMAIN
    HISTORY --> DOMAIN
```

## 확장 포인트

### 새 이벤트 핸들러 추가

```go
// 1. EventHandler 인터페이스 구현
type MyHandler struct{}

func (h *MyHandler) Handle(event *domain.Event) {
    // 처리 로직
}

// 2. ChainHandler에 추가
eventHandler = handler.NewChainHandler(
    logHandler,
    forwardHandler,
    myHandler,  // 새 핸들러
)
```

### 새 저장소 백엔드 추가

```go
// 1. Store 인터페이스 구현
type RedisStore struct{}

func (s *RedisStore) Add(record *Record) { ... }
func (s *RedisStore) Count() int { ... }

// 2. ForwardHandler에 주입
forwardHandler := handler.NewForwardHandler(handler.ForwardHandlerConfig{
    HistoryStore: redisStore,
    ...
})
```

## 기술 스택

| 영역 | 기술 |
|------|------|
| 언어 | Go 1.25+ |
| Slack SDK | [slack-go/slack](https://github.com/slack-go/slack) |
| HTTP | 표준 라이브러리 `net/http` |
| 저장소 | 로컬 JSON 파일 |
| 배포 | 바이너리 / macOS launchd |

## 비기능 요구사항

| 항목 | 사양 |
|------|------|
| 메모리 | ~15-30 MB (일반 사용) |
| 폴링 간격 | 기본 10초 (설정 가능) |
| 히스토리 크기 | 기본 100개 (설정 가능) |
| 타임아웃 | ClickUp API 30초 |
| 재시작 | launchd 자동 재시작 지원 |
