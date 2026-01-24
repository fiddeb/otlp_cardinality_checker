# ui Specification

## Purpose
TBD - created by archiving change add-attribute-catalog. Update Purpose after archive.
## Requirements
### Requirement: Attributes Tab
The system SHALL provide an "Attributes" tab in the web UI for viewing attribute catalog.

#### Scenario: Tab navigation
- **WHEN** user clicks "Attributes" tab
- **THEN** AttributesView component is displayed
- **AND** navigation highlights the active tab

#### Scenario: Tab placement
- **WHEN** viewing the main navigation
- **THEN** "Attributes" tab appears between "Logs" and "Noisy Neighbors"

### Requirement: Attributes Table
The system SHALL display attributes in a sortable table format.

#### Scenario: Table columns
- **WHEN** viewing attributes table
- **THEN** columns are displayed: Key, Cardinality, Count, Sample Values, Signal Types, Scope

#### Scenario: Sortable columns
- **WHEN** user clicks on column header
- **THEN** table is sorted by that column
- **AND** sort indicator (↑↓) is displayed

#### Scenario: Sort direction toggle
- **WHEN** user clicks on already-sorted column
- **THEN** sort direction toggles between ascending and descending

### Requirement: Cardinality Visualization
The system SHALL visualize cardinality levels with color-coded badges.

#### Scenario: High cardinality badge
- **WHEN** attribute has estimated_cardinality > 1000
- **THEN** badge displays as red with text "High"

#### Scenario: Medium cardinality badge
- **WHEN** attribute has estimated_cardinality between 100 and 1000
- **THEN** badge displays as orange with text "Medium"

#### Scenario: Low cardinality badge
- **WHEN** attribute has estimated_cardinality ≤ 100
- **THEN** badge displays as green with text "Low"

### Requirement: Scope Visualization
The system SHALL visualize attribute scope with color-coded badges.

#### Scenario: Resource scope badge
- **WHEN** attribute scope is "resource"
- **THEN** badge displays as blue with text "Resource"

#### Scenario: Attribute scope badge
- **WHEN** attribute scope is "attribute"
- **THEN** badge displays as green with text "Attribute"

#### Scenario: Both scopes badge
- **WHEN** attribute scope is "both"
- **THEN** badge displays as orange with text "Both"

### Requirement: Signal Type Display
The system SHALL display signal types as badges.

#### Scenario: Multiple signal types
- **WHEN** attribute is used in metrics, spans, and logs
- **THEN** three badges are displayed: "metric", "span", "log"

#### Scenario: Single signal type
- **WHEN** attribute is only used in metrics
- **THEN** only "metric" badge is displayed

### Requirement: Sample Values Display
The system SHALL display up to 5 sample values in the table.

#### Scenario: Few samples
- **WHEN** attribute has 3 sample values
- **THEN** all 3 values are displayed comma-separated

#### Scenario: Many samples
- **WHEN** attribute has 10 sample values
- **THEN** first 5 values are displayed
- **AND** "..." indicator shows there are more

### Requirement: Filtering
The system SHALL provide filters for signal type, scope, and minimum cardinality.

#### Scenario: Signal type filter
- **WHEN** user selects "metric" from signal type dropdown
- **THEN** only attributes used in metrics are displayed

#### Scenario: Scope filter
- **WHEN** user selects "resource" from scope dropdown
- **THEN** only resource attributes are displayed

#### Scenario: Minimum cardinality filter
- **WHEN** user enters "1000" in min cardinality input
- **THEN** only attributes with cardinality ≥ 1000 are displayed

#### Scenario: Search filter
- **WHEN** user types "http" in search input
- **THEN** only attributes with keys containing "http" are displayed

#### Scenario: Combined filters
- **WHEN** multiple filters are active
- **THEN** attributes matching ALL filters are displayed

### Requirement: Statistics Bar
The system SHALL display aggregate statistics about the attribute catalog.

#### Scenario: Total attributes stat
- **WHEN** viewing attributes
- **THEN** total count of attributes is displayed

#### Scenario: High cardinality count
- **WHEN** viewing attributes
- **THEN** count of high cardinality attributes (>1000) is displayed

#### Scenario: Resource attributes count
- **WHEN** viewing attributes
- **THEN** count of resource attributes is displayed

### Requirement: Pagination
The system SHALL paginate the attributes list.

#### Scenario: Page size
- **WHEN** more than 50 attributes match filters
- **THEN** only 50 attributes are displayed per page

#### Scenario: Page navigation
- **WHEN** user clicks "Next" or "Previous"
- **THEN** next/previous page of attributes is loaded

#### Scenario: Page indicator
- **WHEN** viewing paginated results
- **THEN** current page and total pages are displayed

### Requirement: Loading States
The system SHALL display loading indicators during data fetching.

#### Scenario: Initial load
- **WHEN** attributes tab is first opened
- **THEN** loading spinner is displayed
- **AND** spinner is removed when data loads

#### Scenario: Filter change
- **WHEN** user changes filters
- **THEN** loading indicator is shown briefly
- **AND** removed when filtered data loads

### Requirement: Error Handling
The system SHALL display user-friendly error messages.

#### Scenario: API error
- **WHEN** API request fails
- **THEN** error message is displayed
- **AND** suggests checking connection or refreshing

#### Scenario: No results
- **WHEN** filters match no attributes
- **THEN** "No attributes found" message is displayed
- **AND** suggests adjusting filters

