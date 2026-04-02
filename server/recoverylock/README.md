# Recovery Lock Bounded Context

This module implements the **Recovery Lock** feature as a bounded context, following
the patterns established in [ADR-0007](../../docs/adr/0007-bounded-context-pattern.md).

## Overview

Recovery Lock is an MDM feature that allows Fleet to set, clear, and rotate a
recovery password on macOS devices with Apple Silicon (ARM). This password is
required to boot the device from recovery mode.

## Architecture

This module follows the **bounded context pattern** with exclusive table ownership:

```
server/recoverylock/
├── recoverylock.go           # DataProviders interface definitions
├── api/                      # Public service interfaces
├── bootstrap/                # Dependency injection wiring
├── internal/                 # Implementation (compiler-enforced boundary)
│   ├── types/               # Internal types & interfaces
│   ├── mysql/               # Database implementation
│   └── service/             # Business logic
└── arch_test.go             # Boundary enforcement tests
```

### Key Principles

1. **Exclusive Table Ownership**: Only `internal/mysql/` accesses `host_recovery_key_passwords`
2. **DataProviders Pattern**: External dependencies injected via explicit interfaces
3. **Public API via `api/` Package**: All external access goes through service methods
4. **`internal/` Enforces Boundaries**: Go compiler prevents imports from outside the module

## Usage

### Initialization

```go
import (
    "github.com/fleetdm/fleet/v4/server/recoverylock/bootstrap"
)

// In serve.go
recoveryLockSvc := bootstrap.New(db, providers)
```

### Getting Recovery Lock Status (Bulk)

```go
// For host listing enrichment
statusMap, err := recoveryLockSvc.GetHostsStatusBulk(ctx, hostUUIDs)
for _, host := range hosts {
    host.RecoveryLockStatus = statusMap[host.UUID]
}
```

### Filtering Hosts by Status

```go
// For host listing filtering (pre-query)
matchingUUIDs, err := recoveryLockSvc.FilterHostsByStatus(ctx, "failed", nil)
// Use matchingUUIDs in WHERE h.uuid IN (...)
```

### Viewing a Password

```go
password, err := recoveryLockSvc.GetHostRecoveryLockPassword(ctx, hostID)
```

### Rotating a Password

```go
err := recoveryLockSvc.RotateHostRecoveryLockPassword(ctx, hostID)
```

## Integration with Host Listing

Host listing uses the **mdmsettingsstatus aggregator** service, which combines
recovery lock status with profiles, declarations, and FileVault status:

```
┌────────────────────────────────────────────────────┐
│              Host Listing (hosts.go)                │
└────────────────────────────────────────────────────┘
                         │
                         ▼
┌────────────────────────────────────────────────────┐
│         mdmsettingsstatus (Aggregator)              │
│  • Combines status from all 4 MDM components        │
│  • Hierarchical aggregation: failed > pending >     │
│    verifying > verified                             │
└────────────────────────────────────────────────────┘
         │                           │
         ▼                           ▼
┌──────────────────┐    ┌────────────────────────────┐
│  recoverylock    │    │   fleet.Datastore (legacy)  │
│  (This Module)   │    │   • Profiles                │
│                  │    │   • Declarations            │
│                  │    │   • FileVault               │
└──────────────────┘    └────────────────────────────┘
```

## Comparison: Current State vs. Bounded Context

### Current State

Today, recovery lock logic is spread across multiple packages with no clear ownership:

- **Business logic** lives in `server/mdm/apple/apple_mdm.go` (cron job, password generation)
- **Result handling** lives in `server/service/apple_mdm.go` (MDM command results)
- **Data access** is 19+ methods on the global `fleet.Datastore` interface, implemented in `server/datastore/mysql/`
- **Types** are defined in `server/fleet/` alongside all other Fleet types
- **Host listing** directly JOINs `host_recovery_key_passwords` from SQL in `hosts.go` and `labels.go`

```go
// Anyone can access the table from anywhere:
ds.GetHostRecoveryLockPassword(ctx, uuid)
ds.SetHostRecoveryLockVerified(ctx, uuid)

// Host listing JOINs recovery lock table directly in SQL:
// SELECT ... LEFT JOIN host_recovery_key_passwords hrkp ON h.uuid = hrkp.host_uuid ...
```

**Characteristics:**
- No code organization — logic scattered across 4+ packages
- 19 methods on the global `fleet.Datastore` interface (alongside 300+ other methods)
- Direct SQL JOINs to `host_recovery_key_passwords` from hosts.go/labels.go
- No ownership boundary — any package can read/write the table
- Difficult to understand the full feature without reading multiple packages

### Bounded Context Approach (`server/recoverylock/`)

The bounded context enforces that **all access** goes through service methods:

```go
// This is the ONLY way to access recovery lock data:
recoveryLockSvc.GetHostRecoveryLockPassword(ctx, hostID)

// Direct table access is impossible from outside:
// - internal/ prevents imports
// - fleet.Datastore has no recovery lock methods
```

**Characteristics:**
- Exclusive table ownership enforced by Go compiler
- Public API via `api/` package only
- Explicit dependencies via DataProviders
- Bulk service calls replace SQL JOINs

### Comparison Table

| Aspect | Current State | Bounded Context |
|--------|--------------|-----------------|
| Code organization | ❌ Scattered across 4+ packages | ✅ One package with clear structure |
| Table ownership | ❌ Anyone can access | ✅ Exclusive (`internal/` enforced) |
| Interface | Global `fleet.Datastore` | Public `api.Service` |
| Dependencies | Implicit | Explicit (`DataProviders`) |
| JOINs to table | Direct SQL | Via bulk service calls |
| Enforcement | None | Compiler + `arch_test.go` |
| Testing | Requires mock of full Datastore | Can mock DataProviders |
| Discoverability | Poor — must know where to look | Good |

### Pros of Bounded Context

1. **True encapsulation**: Table schema can change without affecting callers
2. **Clear contracts**: Public API is explicit and documented
3. **Testability**: DataProviders can be mocked independently
4. **Evolvability**: Internal implementation can be refactored freely
5. **Discoverability**: All recovery lock code in one place
6. **Prevents accidental coupling**: Can't accidentally query the table

### Cons of Bounded Context

1. **More code structure**: bootstrap/, api/, internal/ directories
2. **Performance overhead**: Bulk queries instead of JOINs (mitigated by index)
3. **Learning curve**: DataProviders pattern is less familiar
4. **Migration effort**: Need to move 19 methods, update all callers
5. **Two queries**: Filter then fetch (vs single JOIN query)

## Database Schema

This module owns the `host_recovery_key_passwords` table:

```sql
CREATE TABLE `host_recovery_key_passwords` (
    `host_uuid` varchar(255) NOT NULL,              -- Primary key, foreign to hosts.uuid
    `encrypted_password` blob NOT NULL,             -- Active password (encrypted)
    `status` varchar(20) DEFAULT NULL,              -- NULL|'pending'|'verifying'|'verified'|'failed'
    `operation_type` varchar(20) NOT NULL,          -- 'install'|'remove'
    `error_message` text,                           -- Error detail for install/remove operations
    `deleted` tinyint(1) NOT NULL DEFAULT '0',      -- Soft delete flag
    `created_at` timestamp(6) DEFAULT CURRENT_TIMESTAMP(6),
    `updated_at` timestamp(6) DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    `pending_encrypted_password` blob,              -- New password during rotation
    `pending_error_message` text,                   -- Error for rotation operation
    `auto_rotate_at` timestamp(6) NULL,             -- When to auto-rotate viewed password
    PRIMARY KEY (`host_uuid`),
    KEY `status` (`status`),
    KEY `operation_type` (`operation_type`),
    KEY `deleted` (`deleted`),
    KEY `idx_auto_rotate_at` (`auto_rotate_at`)
);
```

## Testing

```bash
# Unit tests
go test ./server/recoverylock/...

# Integration tests
MYSQL_TEST=1 REDIS_TEST=1 go test ./server/recoverylock/...

# Boundary enforcement
go test ./server/recoverylock/arch_test.go
```
