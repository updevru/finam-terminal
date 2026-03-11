# Plan: Редизайн списка счетов — двухстрочный формат

## Overview

Переделка таблицы счетов с трёхстолбцового формата на двухстрочный: ID на первой строке, Equity + PnL на второй. Убирается столбец Type, добавляется цветная индикация UnrealizedPnL.

## Phase 1: Number Formatting Utility

- [x] Task: Write tests for thousand-separator number formatting `1795a7c`
  - Acceptance: Tests cover positive, negative, zero, large numbers, invalid input
- [x] Task: Implement `formatNumber` utility with space as thousand separator `a564be8`
  - Acceptance: All formatting tests pass

## Phase 2: Two-Row Account Rendering

- [ ] Task: Write tests for two-row account rendering logic
  - Acceptance: Tests verify correct row mapping (account index → table row), cell content, and colors
- [ ] Task: Refactor `updateAccountList` to render 2 rows per account
  - Acceptance: Each account renders as ID row + Equity/PnL row, no Type column, colors correct
- [ ] Task: Handle error accounts in two-row format
  - Acceptance: LoadError accounts show ID + "[error]" on second row in red

## Phase 3: Navigation & Selection

- [ ] Task: Write tests for account-index-to-row mapping
  - Acceptance: Tests verify `accountIdxToRow` and `rowToAccountIdx` conversions
- [ ] Task: Update Up/Down navigation to skip by 2 rows
  - Acceptance: Arrow keys navigate between accounts, not individual rows
- [ ] Task: Update selection highlight to cover both rows of selected account
  - Acceptance: Both rows of selected account are visually highlighted

## Phase 4: Testing & Polish

- [ ] Task: Integration test — full render cycle with multiple accounts
  - Acceptance: Render with 0, 1, and 3+ accounts produces correct table structure
- [ ] Task: Manual verification and visual polish
  - Acceptance: Layout looks clean within 30-char panel, no overflow issues
