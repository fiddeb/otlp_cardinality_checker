# Spec Delta: storage

## ADDED Requirements

### Requirement: Deep Watch Interface Methods
The `Storage` interface SHALL provide operations to activate, deactivate, and query Deep Watch state for attribute keys.

#### Description
The storage layer MUST expose four methods for managing watched attributes. Implementations MUST be safe for concurrent access. The in-memory implementation MUST maintain a separate `watched` map that does not interfere with the existing `attributes` map.

#### Requirements
1. `WatchAttribute(ctx, key)` MUST activate Deep Watch for the given key and initialize a new `WatchedAttribute`
2. `WatchAttribute` MUST return an error if the maximum watched fields limit is already reached
3. `WatchAttribute` MUST be idempotent: calling it on an already-watched key MUST return nil and not reset existing data
4. `UnwatchAttribute(ctx, key)` MUST deactivate Deep Watch and discard all collected values for the key
5. `GetWatchedAttribute(ctx, key)` MUST return the `WatchedAttribute` for a watched key, or nil if not watched
6. `ListWatchedAttributes(ctx)` MUST return all currently active `WatchedAttribute` entries
7. `StoreAttributeValue` MUST be updated to additionally call `WatchedAttribute.AddValue` when the key is watched

#### Scenario: Activate and collect
**GIVEN** storage is initialized  
**WHEN** `WatchAttribute(ctx, "workflow.folder")` is called  
**AND** `StoreAttributeValue(ctx, "workflow.folder", "reports/q1", "metric", "attribute")` is called  
**THEN** `GetWatchedAttribute(ctx, "workflow.folder")` returns a non-nil `WatchedAttribute`  
**AND** `WatchedAttribute.Values["reports/q1"]` equals 1

#### Scenario: Idempotent activation
**GIVEN** `workflow.folder` is already watched with 100 observations  
**WHEN** `WatchAttribute(ctx, "workflow.folder")` is called again  
**THEN** the existing data is preserved  
**AND** `WatchedAttribute.TotalObservations` remains 100

#### Scenario: Deactivate clears data
**GIVEN** `workflow.folder` is watched with collected values  
**WHEN** `UnwatchAttribute(ctx, "workflow.folder")` is called  
**THEN** `GetWatchedAttribute(ctx, "workflow.folder")` returns nil  
**AND** the values map is freed from memory

#### Scenario: Limit enforced
**GIVEN** 10 keys are currently watched (max limit)  
**WHEN** `WatchAttribute(ctx, "new.key")` is called  
**THEN** an error is returned  
**AND** the existing 10 watched keys are unaffected

#### Scenario: StoreAttributeValue hot path
**GIVEN** `workflow.folder` is NOT watched  
**WHEN** `StoreAttributeValue(ctx, "workflow.folder", "v", "metric", "attribute")` is called  
**THEN** no watch-related work occurs (read-lock check only)  
**AND** existing `AttributeMetadata` is updated normally

#### Scenario: Inactive watch does not collect
**GIVEN** `workflow.folder` watch is present but `Active = false` (restored from session)  
**WHEN** `StoreAttributeValue(ctx, "workflow.folder", "new-value", "metric", "attribute")` is called  
**THEN** `WatchedAttribute.Values` is NOT updated  
**AND** `TotalObservations` is NOT incremented
