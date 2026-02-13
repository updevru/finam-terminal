---
description: Show project progress and track status
allowed-tools: Bash(ls*), Bash(git status*), Read
---

# Conductor Status

Display current project state and track progress.

## Output Format

```
📊 Conductor Status

Project: [Name from product.md]
Stack: [Key technologies]

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🎯 Active Track: track_20241230_001
   Title: Add dark mode toggle
   Phase: 2/3
   Progress: 6/9 tasks complete (67%)
   Current: Phase 2, Task 2 [~]

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📋 Recent Tracks (last 5):
1. track_20241230_001 - In Progress - Dark mode toggle
2. track_20241229_003 - ✅ Complete - User authentication
3. track_20241229_002 - ✅ Complete - Database setup
4. track_20241229_001 - ✅ Complete - Initial project setup

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Next Action:
→ Continue: /conductor:implement
→ New Track: /conductor:track "Feature name"

Last updated: 2024-12-30 15:52:30
```

## Process

1. **Read tracks.md** for overview
2. **Find active track** (if any)
3. **Parse plan.md** for active track
4. **Count completed tasks**
5. **Show concise summary**

Keep it brief and actionable!

## Implementation Details

1. Read `conductor/tracks.md` to get all tracks
2. For each track, read its `plan.md` to count tasks
3. Calculate progress percentage
4. Identify active track (marked with `[~]`)
5. Display formatted status report
