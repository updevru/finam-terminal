---
description: Create a new feature or bug fix track with spec and plan
argument-hint: Optional feature/bug description
allowed-tools: Bash(mkdir*), Bash(ls*), Read, Write, Edit, AskUserQuestion
---

# Create New Track

Start a new development track with specification and implementation plan.

## Usage
```
/conductor:track "Add dark mode toggle"
```

Or run without argument for interactive mode.

## Process

### Step 1: Verify Setup

```bash
test -d conductor/ || {
  echo "❌ Conductor not initialized. Run /conductor:setup first"
  exit 1
}
```

### Step 2: Get Track Description

If not provided:
"What feature or bug are you working on? (Be specific)"

### Step 3: Generate Track ID

```bash
TRACK_ID="track_$(date +%Y%m%d)_$(printf '%03d' $(($(ls conductor/tracks/ | wc -l) + 1)))"
# Example: track_20241230_001
```

### Step 4: Create Track Directory

```bash
mkdir -p conductor/tracks/${TRACK_ID}
```

### Step 5: Generate Specification (Interactive)

**Ask 5-7 key questions:**

1. "What problem does this solve? (User perspective)"
2. "What should the feature do? (Core functionality)"
3. "Any technical constraints?" (APIs, performance, etc.)
4. "How do you verify success?" (Acceptance criteria)
5. "Any edge cases to handle?"

**Generate:** `conductor/tracks/${TRACK_ID}/spec.md`

```markdown
# Spec: [Feature Name]

## Problem
[User problem statement]

## Solution
[What we're building]

## Requirements
- Functional requirement 1
- Functional requirement 2
- Technical requirement 1

## Acceptance Criteria
- [ ] Criterion 1
- [ ] Criterion 2

## Edge Cases
- Edge case 1
- Edge case 2

## Dependencies
- Library X
- API Y (if applicable)
```

### Step 6: Generate Implementation Plan

Based on spec + workflow.md:

**Phase structure:**
```markdown
# Plan: [Feature Name]

## Overview
[2-3 sentence summary]

## Phase 1: Foundation
- [ ] Task: Set up basic structure
  - Acceptance: Structure exists, tests pass
- [ ] Task: Implement core logic
  - Acceptance: Core function works
- [ ] Task: Add error handling
  - Acceptance: Errors handled gracefully

## Phase 2: Integration
- [ ] Task: Connect to external API
  - Acceptance: API calls work
- [ ] Task: Add UI component (if applicable)
  - Acceptance: Component renders
- [ ] Task: Add validation
  - Acceptance: Invalid input rejected

## Phase 3: Testing & Polish
- [ ] Task: Add comprehensive tests
  - Acceptance: 80%+ coverage
- [ ] Task: Update documentation
  - Acceptance: README updated
- [ ] Task: Manual verification
  - Acceptance: Feature works end-to-end
```

**If TDD workflow:** Inject "Write test" sub-tasks

**Generate:** `conductor/tracks/${TRACK_ID}/plan.md`

### Step 7: Create Metadata

```json
{
  "id": "track_20241230_001",
  "title": "Add dark mode toggle",
  "type": "feature",
  "status": "planned",
  "created_at": "2024-12-30T15:45:00Z",
  "phases": 3,
  "total_tasks": 9
}
```

### Step 8: Update Tracks Index

Append to `conductor/tracks.md`:

```markdown
### Track: track_20241230_001
- **Title**: Add dark mode toggle
- **Status**: Planned
- **Created**: 2024-12-30
- **Phases**: 3 phases, 9 tasks
- **Active**: No
```

### Step 9: Present to User

```
✅ Track created: track_20241230_001

📋 Spec: conductor/tracks/track_20241230_001/spec.md
📝 Plan: conductor/tracks/track_20241230_001/plan.md

Next: Review the plan. When ready, run:
/conductor:implement track_20241230_001
```

## No Auto-Implementation

**Important:** Do NOT start implementing. Wait for user to review and approve.
