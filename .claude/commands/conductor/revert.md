---
description: Revert a track, phase, or task using git
argument-hint: Optional track ID to revert
allowed-tools: Bash(git log*), Bash(git revert*), Bash(git diff*), Read, Write, Edit, AskUserQuestion
---

# Revert Track Work

Git-aware revert that understands tracks, phases, and tasks.

## Usage
```
/conductor:revert                  # Interactive selection
/conductor:revert track_001        # Revert specific track
```

## Process

### Step 1: Interactive Selection

```
What would you like to revert?

In Progress:
1. track_20241230_001 - Dark mode toggle (Phase 2, 6/9 tasks)

Recently Completed:
2. track_20241229_003 - User authentication
3. track_20241229_002 - Database setup
4. track_20241229_001 - Initial setup

Select (1-4) or 'cancel':
```

### Step 2: Granularity Choice

```
Revert track_20241230_001:

1. Entire track (all 6 tasks)
2. Current phase only (Phase 2, 3 tasks)
3. Last task only (Task 2)

Select (1-3):
```

### Step 3: Find Git Commits

```bash
# Search for commits with track ID
git log --oneline --grep="track_20241230_001" --since="7 days ago"

# Example output:
abc1234 docs(track_20241230_001): Update plan status
def5678 feat(track_20241230_001): Implement toggle component
ghi9012 feat(track_20241230_001): Add theme context
```

### Step 4: Show What Will Be Reverted

```
This will revert 6 commits:

Commits to revert:
- abc1234 docs: Update plan status
- def5678 feat: Implement toggle component
- ghi9012 feat: Add theme context
- jkl3456 test: Add toggle tests
- mno7890 feat: Add theme provider
- pqr1234 feat: Set up theme structure

Files affected:
- src/theme/ThemeContext.tsx
- src/components/Toggle.tsx
- src/theme/ThemeProvider.tsx
- tests/Toggle.test.tsx
- conductor/tracks/track_20241230_001/plan.md

⚠️  WARNING: This cannot be easily undone!
```

### Step 5: Confirm Multiple Times

```
Are you SURE you want to revert this track? (yes/no): yes

This will PERMANENTLY revert 6 commits. Type 'REVERT' to confirm: REVERT
```

### Step 6: Execute Revert

```bash
# Revert commits in reverse order (newest first)
git revert --no-edit pqr1234
git revert --no-edit mno7890
git revert --no-edit jkl3456
git revert --no-edit ghi9012
git revert --no-edit def5678
git revert --no-edit abc1234
```

### Step 7: Update Plan

Reset tasks in `plan.md`:
```markdown
- [x] Task 2 (def5678)  →  - [ ] Task 2
```

Commit update:
```bash
git commit -am "docs(track_20241230_001): Reset plan after revert"
```

### Step 8: Update Tracks

```markdown
### Track: track_20241230_001
- **Status**: Reverted → Planned
- **Progress**: Reset to 0/9 tasks
```

### Step 9: Final Message

```
✅ Reverted track_20241230_001

6 commits reverted successfully.
Plan reset to initial state.

You can:
- Resume: /conductor:implement track_20241230_001
- Delete: rm -rf conductor/tracks/track_20241230_001
- Start fresh: /conductor:track "New feature"
```

## Safety Notes

- Uses `git revert` (preserves history)
- NEVER uses `git reset` (destructive)
- Multiple confirmations required
- Shows exactly what will change
