# Plan: Редизайн списка счетов — двухстрочный формат

## Overview

Переделка таблицы счетов с трёхстолбцового формата на двухстрочный: ID на первой строке, Equity + PnL на второй. Убирается столбец Type, добавляется цветная индикация UnrealizedPnL.

## Phase 1: Number Formatting Utility [checkpoint: bb98200]

- [x] Task: Write tests for thousand-separator number formatting `1795a7c`
  - Acceptance: Tests cover positive, negative, zero, large numbers, invalid input
- [x] Task: Implement `formatNumber` utility with space as thousand separator `a564be8`
  - Acceptance: All formatting tests pass

## Phase 2: Two-Row Account Rendering [checkpoint: 058da99]

- [x] Task: Write tests for two-row account rendering logic `724f927`
  - Acceptance: Tests verify correct row mapping (account index → table row), cell content, and colors
- [x] Task: Refactor `updateAccountList` to render 2 rows per account `11b3304`
  - Acceptance: Each account renders as ID row + Equity/PnL row, no Type column, colors correct
- [x] Task: Handle error accounts in two-row format `c02152e`
  - Acceptance: LoadError accounts show ID + "[error]" on second row in red

## Phase 3: Navigation & Selection [checkpoint: 16b11ec]

- [x] Task: Write tests for account-index-to-row mapping `bfbd253`
  - Acceptance: Tests verify `accountIdxToRow` and `rowToAccountIdx` conversions
- [x] Task: Update Up/Down navigation to skip by 2 rows `b777043`
  - Acceptance: Arrow keys navigate between accounts, not individual rows
- [x] Task: Update selection highlight to cover both rows of selected account `96e08de`
  - Acceptance: Both rows of selected account are visually highlighted

## Phase 4: Testing & Polish

- [x] Task: Integration test — full render cycle with multiple accounts `4521528`
  - Acceptance: Render with 0, 1, and 3+ accounts produces correct table structure
- [~] Task: Manual verification and visual polish
  - Acceptance: Layout looks clean within 30-char panel, no overflow issues
