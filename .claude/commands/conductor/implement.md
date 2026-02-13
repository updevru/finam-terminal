---
description: Execute the implementation plan for a track
argument-hint: Optional track ID
allowed-tools: Bash(git*), Bash(npm test*), Bash(python*), Read, Write, Edit, AskUserQuestion, TodoWrite, mcp__context7__*
---

# Implement Track

Execute tasks from a track's plan using TDD workflow.

## Usage
```
/conductor:implement               # Implement active track
/conductor:implement track_001     # Implement specific track
```

## Process

### Step 1: Load Track

```bash
# If no argument, find active track
if [ -z "$1" ]; then
  TRACK=$(grep -A 5 "Active: Yes" conductor/tracks.md | head -1 | cut -d: -f2)
else
  TRACK="$1"
fi

test -f "conductor/tracks/${TRACK}/plan.md" || {
  echo "❌ Track not found. Run /conductor:status to see available tracks"
  exit 1
}
```

### Step 2: Invoke Skills

**Skills auto-load:**
- `conductor-context` → Loads spec, plan, workflow
- `conductor-workflow` → Enforces task lifecycle

### Step 3: Execute Tasks One-by-One

The `conductor-workflow` skill handles:
1. Find next `[ ]` task
2. Mark `[~]` in-progress
3. Execute TDD cycle
4. Commit with metadata
5. Mark `[x]` with SHA
6. Commit plan update
7. Check if phase complete

### Step 4: Phase Checkpoints

After each phase:
```
✅ Phase 1 complete (3/3 tasks)

Please verify:
- Run app and test feature manually
- Check all tests pass
- Review code quality

Ready to proceed to Phase 2? (yes/no)
```

### Step 5: Track Completion

After final task:
```
🎉 Track complete: track_20241230_001

All 9 tasks completed across 3 phases.

Update tracks.md status?
- [x] Mark as Complete
- [ ] Archive track
- [ ] Delete track (not recommended)
```

Update `conductor/tracks.md`:
```markdown
### Track: track_20241230_001
- **Title**: Add dark mode toggle
- **Status**: ✅ Completed
- **Completed**: 2024-12-30
```

## Important

- ONE task at a time
- Get user approval between phases
- NEVER skip testing
- ALWAYS commit with track metadata

## Workflow Details

This command delegates the actual task execution to the `conductor-workflow` skill, which:

1. **Pre-work**: Check git, find next task, mark in-progress
2. **TDD Cycle**: Write test → Fail → Implement → Pass
3. **Commit**: Stage changes, commit with track metadata
4. **Update Plan**: Mark task complete with commit SHA
5. **Verify**: Run full test suite at phase end
6. **Checkpoint**: Ask for user approval between phases
