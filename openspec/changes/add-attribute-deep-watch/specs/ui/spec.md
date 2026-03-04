# Spec Delta: ui

## MODIFIED Requirements

### Requirement: Attribute Catalog Watch Toggle
The Attribute Catalog table SHALL display a watch toggle for each attribute row, enabling users to activate or deactivate Deep Watch without leaving the UI.

#### Description
Each row in the Attribute Catalog MUST render a toggle button in a dedicated **Watch** column. The toggle MUST reflect live watch state from the API. Activating a watch MUST call `POST /api/v1/attributes/:key/watch`; deactivating MUST call `DELETE /api/v1/attributes/:key/watch`.

#### Requirements
1. The Attribute Catalog table MUST include a **Watch** column as the last column
2. Each row MUST render a toggle button showing active (`watching`) or inactive state
3. An active watch row MUST display a `"watching since HH:MM"` badge next to the attribute key
4. When the maximum watched fields limit is reached, inactive toggles MUST be disabled with a tooltip explaining the limit
5. Activating a watch MUST optimistically update the row to active state and confirm on API response
6. An overflow state MUST be indicated with a warning icon in the watch badge with tooltip `"Value cap reached — collection paused"`

#### Scenario: Toggle activates watch
**GIVEN** `workflow.folder` is not watched  
**WHEN** user clicks the watch toggle on the `workflow.folder` row  
**THEN** `POST /api/v1/attributes/workflow.folder/watch` is called  
**AND** the row badge shows `watching since HH:MM`  
**AND** the key becomes clickable to open Value Explorer

#### Scenario: Toggle deactivates watch
**GIVEN** `workflow.folder` is being watched  
**WHEN** user clicks the active watch toggle  
**THEN** `DELETE /api/v1/attributes/workflow.folder/watch` is called  
**AND** the row returns to inactive state  
**AND** the badge is removed

#### Scenario: Limit reached disables toggles
**GIVEN** 10 keys are actively watched  
**WHEN** the Attribute Catalog renders  
**THEN** all inactive toggles are disabled  
**AND** hovering shows tooltip `"Maximum 10 fields can be watched simultaneously"`

#### Scenario: Overflow indicator
**GIVEN** `workflow.folder` has overflowed (10,000 unique values reached)  
**WHEN** the Attribute Catalog renders  
**THEN** the watch badge for `workflow.folder` shows a warning icon  
**AND** hovering shows tooltip `"Value cap reached — collection paused"`

## ADDED Requirements

### Requirement: Value Explorer Panel
The system SHALL provide a Value Explorer panel that opens when a user clicks on a watched attribute key, showing all collected values with counts.

#### Description
The Value Explorer MUST be accessible by clicking on the attribute key name in the Attribute Catalog when Deep Watch is active for that key. It MUST provide search and sort capabilities. The panel MUST fetch data from `GET /api/v1/attributes/:key/watch`.

#### Requirements
1. Clicking a watched attribute key MUST open the Value Explorer
2. The Value Explorer MUST display: `key`, `watching since`, `unique_count`, `total_observations`, and a message if `overflow = true`
3. The Value Explorer MUST render a table with columns **Value** and **Count**
4. The table MUST be sorted by count descending by default
5. The Value Explorer MUST include a search input that filters rows by value prefix (client-side)
6. The Value Explorer MUST include a **Refresh** button that re-fetches data from the API
7. The Value Explorer MUST display a message if no values have been collected yet (`"No values collected yet. Watching since HH:MM."`)
8. Clicking a watched attribute key when `overflow = true` MUST display a prominent warning banner

#### Scenario: Open Value Explorer
**GIVEN** `workflow.folder` is watched with 847 unique values  
**WHEN** user clicks on `workflow.folder` in the Attribute Catalog  
**THEN** the Value Explorer opens  
**AND** shows `unique_count: 847`  
**AND** renders the value table sorted by count descending

#### Scenario: Search filters table
**GIVEN** the Value Explorer is open for `workflow.folder`  
**WHEN** user types `reports` in the search input  
**THEN** only rows where the value starts with `reports` are visible

#### Scenario: Empty state
**GIVEN** `workflow.folder` was just activated (no telemetry received yet)  
**WHEN** user clicks on `workflow.folder`  
**THEN** the Value Explorer shows `"No values collected yet. Watching since HH:MM."`

#### Scenario: Overflow banner
**GIVEN** `workflow.folder` has `overflow = true`  
**WHEN** Value Explorer is opened  
**THEN** a warning banner reads `"Value cap of 10,000 reached. Some values may not appear."`
