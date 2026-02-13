---
name: conductor-context
description: Loads Conductor project context automatically. Use when user asks about the project, starts tracks, implements features, or needs project info. Auto-loads conductor/ files as needed.
---

# Conductor Context Skill

## When I activate
- User mentions "project", "our app", "what are we building"
- User runs /conductor:setup, /conductor:track, or /conductor:implement
- User asks about tech stack, workflow, or style guides
- Need context for generating specs or plans

## What I do

### 1. Check if Conductor is initialized

First, verify the project has Conductor set up:

```bash
test -d conductor/ && echo "initialized" || echo "not initialized"
```

If not initialized:
- Tell user: "Run `/conductor:setup` first to initialize Conductor"
- STOP here

### 2. Load only what's needed for current task

**For planning/new tracks:**
- Read `conductor/product.md` (2-3 sentence summary only)
- Read `conductor/tech-stack.md` (list technologies only)

**For implementation:**
- Read `conductor/workflow.md` (full content)
- Read active track from `conductor/tracks.md`
- Read `conductor/tracks/<id>/spec.md`
- Read `conductor/tracks/<id>/plan.md`

**For style questions:**
- List available guides in `conductor/styleguides/`
- Read specific guide only if asked

### 3. Provide concise summary

Format:
```
Project: [1 sentence from product.md]
Stack: [key technologies]
Active Track: [ID and title, if any]
Current Task: [Phase X, Task Y, if implementing]
```

## Token Efficiency Rules

- **Never** load all files at once
- **Never** reproduce entire file contents
- **Always** summarize briefly (2-3 sentences max)
- **Only** load what's relevant to current conversation
- If brownfield: Read only package.json, README.md, and file tree

## Important

This skill caches context in the conversation. Don't reload unless files change.
