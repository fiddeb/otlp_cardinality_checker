# Spec Delta: Runtime Modes

## ADDED Requirements

### Requirement: Runtime Mode Selection

**ID**: `runtime-modes.mode-selection`  
**Status**: Draft  
**Priority**: High

#### Description
The system SHALL support multiple runtime modes selectable at startup via command-line flags or environment variables. The minimal mode MUST disable UI components and use memory-only storage.

#### Requirements
1. System SHALL accept `--minimal` or `--cicd` flag to enable minimal mode
2. System SHALL accept `OCC_MINIMAL` environment variable as alternative
3. CLI flags MUST take precedence over environment variables
4. System SHALL log the active mode at startup
5. Mode MUST NOT be changeable after startup without restart

#### Scenario: Start in minimal mode via CLI
**GIVEN** OCC is not running  
**WHEN** user executes `occ start --minimal`  
**THEN** OCC SHALL start in minimal mode  
**AND** UI server SHALL NOT be initialized  
**AND** OTLP receiver SHALL be initialized  
**AND** API server SHALL be initialized  
**AND** log MUST contain "Running in minimal mode"

#### Scenario: Start in minimal mode via environment
**GIVEN** environment variable `OCC_MINIMAL=true`  
**AND** OCC is not running  
**WHEN** user executes `occ start`  
**THEN** OCC SHALL start in minimal mode  
**AND** behavior SHALL be identical to CLI flag

#### Scenario: CLI overrides environment
**GIVEN** environment variable `OCC_MINIMAL=false`  
**WHEN** user executes `occ start --minimal`  
**THEN** OCC SHALL start in minimal mode (CLI flag wins)

#### Scenario: Default mode is normal
**GIVEN** no minimal mode flag is set  
**AND** no `OCC_MINIMAL` environment variable is set  
**WHEN** user executes `occ start`  
**THEN** OCC SHALL start in normal mode  
**AND** all components SHALL be initialized (OTLP, API, UI, storage)

---

### Requirement: Component Lifecycle by Mode

**ID**: `runtime-modes.component-lifecycle`  
**Status**: Draft  
**Priority**: High

#### Description
The system SHALL selectively initialize components based on the active runtime mode. In minimal mode, only OTLP receivers and API server MUST be active.

#### Requirements
1. Normal mode SHALL initialize: OTLP receivers, API server, UI server, memory storage with session save/load
2. Minimal mode SHALL initialize: OTLP receivers, API server, memory-only storage
3. Minimal mode MUST NOT initialize UI server or UI-related components
4. Minimal mode MUST NOT enable session save/load functionality
5. Minimal mode MUST use bounded memory storage with eviction policy
6. Component initialization order MUST be: storage → OTLP → API → UI (if normal)

#### Scenario: Normal mode initializes all components
**GIVEN** no minimal mode flag is set  
**WHEN** OCC starts  
**THEN** OTLP gRPC receiver SHALL be listening  
**AND** OTLP HTTP receiver SHALL be listening  
**AND** API server SHALL be listening  
**AND** UI server SHALL be listening  
**AND** memory storage with session capability SHALL be initialized

#### Scenario: Minimal mode skips UI
**GIVEN** `--minimal` flag is set  
**WHEN** OCC starts  
**THEN** OTLP receivers SHALL be listening  
**AND** API server SHALL be listening  
**AND** UI server SHALL NOT be listening  
**AND** UI server port MUST NOT be bound  
**AND** session save/load commands SHALL be unavailable

---

### Requirement: Auto-Shutdown with Duration

**ID**: `runtime-modes.auto-shutdown`  
**Status**: Draft  
**Priority**: High

#### Description
The system SHALL support automatic shutdown after a specified duration in minimal mode, optionally generating a report before exit.

#### Requirements
1. System SHALL accept `--duration` flag with Go duration format (e.g., "5m", "1h")
2. Duration timer MUST start immediately after successful startup
3. On timer expiration, system MUST initiate graceful shutdown
4. Shutdown MUST stop accepting new OTLP data first
5. Shutdown MUST drain in-flight requests (5 second grace period)
6. Report generation MUST complete before final exit
7. If `--api-only` is set, duration timer MUST NOT trigger shutdown

#### Scenario: Auto-shutdown after duration
**GIVEN** OCC started with `--minimal --duration 1m`  
**WHEN** 60 seconds have elapsed  
**THEN** OCC SHALL stop accepting new OTLP connections  
**AND** OCC SHALL drain existing requests  
**AND** OCC SHALL generate report  
**AND** OCC SHALL exit with appropriate code

#### Scenario: API-only mode ignores duration
**GIVEN** OCC started with `--minimal --duration 1m --api-only`  
**WHEN** 60 seconds have elapsed  
**THEN** OCC SHALL remain running  
**AND** API SHALL remain accessible

#### Scenario: Signal cancels duration timer
**GIVEN** OCC started with `--minimal --duration 5m`  
**AND** 2 minutes have elapsed  
**WHEN** SIGTERM signal is received  
**THEN** duration timer SHALL be cancelled  
**AND** graceful shutdown SHALL proceed immediately

---

### Requirement: Graceful Shutdown

**ID**: `runtime-modes.graceful-shutdown`  
**Status**: Draft  
**Priority**: Critical

#### Description
The system MUST shut down gracefully, ensuring no data loss and proper resource cleanup, whether triggered by duration timeout or OS signal.

#### Requirements
1. System MUST handle SIGTERM and SIGINT signals
2. On shutdown trigger, OTLP receivers MUST stop accepting new connections
3. In-flight OTLP requests MUST be processed or gracefully rejected
4. Shutdown grace period MUST be at least 5 seconds, max 30 seconds
5. Context cancellation MUST propagate to all active goroutines
6. All file handles and network connections MUST be closed
7. Report generation MUST NOT be interrupted unless context deadline exceeded

#### Scenario: Clean shutdown on SIGTERM
**GIVEN** OCC is running in minimal mode  
**AND** OTLP data is being received  
**WHEN** SIGTERM signal is received  
**THEN** new OTLP connections SHALL be refused  
**AND** active OTLP requests SHALL complete or timeout after 5s  
**AND** report SHALL be generated  
**AND** all resources SHALL be closed  
**AND** process SHALL exit within 30 seconds

#### Scenario: Forced shutdown after grace period
**GIVEN** OCC is running in minimal mode  
**AND** shutdown has been triggered  
**AND** 30 seconds have elapsed  
**WHEN** grace period expires  
**THEN** OCC SHALL force-exit  
**AND** log SHALL contain warning about forced shutdown

---

### Requirement: Resource Limits

**ID**: `runtime-modes.resource-limits`  
**Status**: Draft  
**Priority**: High

#### Description
The system SHALL enforce configurable resource limits in minimal mode to prevent unbounded memory growth in CI/CD environments.

#### Requirements
1. System SHALL accept `--max-memory` flag (in MB)
2. Default max memory MUST be 512MB
3. System SHOULD monitor actual memory usage periodically
4. System MUST log warning at 80% memory threshold
5. System MUST evict oldest data when memory limit reached
6. Eviction MUST preserve cardinality accuracy for remaining data

#### Scenario: Memory limit enforced
**GIVEN** OCC started with `--minimal --max-memory 256`  
**WHEN** memory usage reaches 250MB (>80%)  
**THEN** warning SHALL be logged  
**AND** oldest metrics SHALL be evicted  
**AND** memory usage SHALL remain below 256MB

#### Scenario: Memory tracking accurate
**GIVEN** OCC is running in minimal mode  
**WHEN** OTLP data is ingested  
**THEN** memory usage SHALL be tracked  
**AND** memory usage SHALL be queryable via API (/health endpoint)  
**AND** memory usage SHALL reflect actual heap allocation within 10% accuracy
