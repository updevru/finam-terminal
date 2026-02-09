# Specification: Portfolio Tabs (History & Orders)

## Overview
This track introduces a tabbed interface within the primary "Positions" window. This allows users to switch between the current **Positions** view, a new **History** view (trade history), and a new **Orders** view (active/pending orders) using keyboard shortcuts.

## Functional Requirements
- **Tabbed Layout:**
    - Replace the static "Positions" title/view with a tabbed interface.
    - Tab headers: `Positions`, `History`, `Orders`.
    - Visual Style: A solid background bar with the active tab highlighted in a different color.
- **Navigation:**
    - **Left/Right Arrows:** Switch between adjacent tabs.
    - **Tab / Shift+Tab:** Cycle through tabs (forward/backward).
- **History View:**
    - Displays a table of completed trades for the selected account.
    - Columns: Operation (Buy/Sell), Instrument, Quantity, Price per Unit, Total Amount, Date and Time.
- **Orders View:**
    - Displays a table of non-executed (pending/active) orders.
    - Columns: Instrument, Mode, Type, Requested, Activation, Price, Time, Executed, Duration (Expiry), Status.
- **Data Management:**
    - **Refresh on Entry:** Data for History and Orders is fetched when the user switches to the respective tab.
    - **Manual Refresh:** Pressing 'R' triggers a manual data refresh for the active tab.

## Non-Functional Requirements
- **UI Consistency:** Tables in History and Orders must match the styling (borders, alignment, colors) of the existing Positions table.
- **Performance:** Switching tabs should be responsive; loading states should be shown if API calls take time.

## Acceptance Criteria
- [ ] Users can switch between three tabs using arrows and Tab keys.
- [ ] The History tab correctly displays past transactions with all required columns.
- [ ] The Orders tab correctly displays active/pending orders with all required columns.
- [ ] The active tab is clearly visually distinguished.
- [ ] Pressing 'R' updates the data in the current tab.

## Out of Scope
- Modifying or canceling orders from the Orders tab (this will be handled in a future track).
- Advanced filtering or searching within the History/Orders tables.
