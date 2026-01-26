# Specification: full_width_positions_table

## Overview
Currently, the Positions table in the terminal UI has a fixed width. On wide terminal windows, this results in the table occupying only a portion of the available space, leaving the rest of the "Positions" section empty. This track aims to modify the UI so that the Positions table automatically expands to occupy the full width of its container.

## Functional Requirements
- Modify the `tview.Table` configuration for the Positions view to enable expansion.
- Ensure all columns grow proportionally to fill the available horizontal space.
- The table must remain responsive to terminal window resizing.

## Non-Functional Requirements
- Maintain existing data rendering logic and styling.
- Ensure no regression in table scrolling or selection behavior.

## Acceptance Criteria
- [ ] The Positions table occupies 100% of the horizontal width of its parent container.
- [ ] Resizing the terminal window causes the table to adjust its width accordingly.
- [ ] All columns scale proportionally without manual width hardcoding.

## Out of Scope
- Modifying other tables (Quotes, Orders, etc.) unless necessary for shared component updates.
- Changing the content or data source of the Positions table.
