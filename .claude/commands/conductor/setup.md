---
description: Initialize Conductor in your project with interactive setup
argument-hint: Optional project description
allowed-tools: Bash(mkdir*), Bash(git*), Read, Write, Edit
---

# Conductor Setup

Initialize Conductor's context-driven development workflow.

## What this does
1. Detect if project is new (greenfield) or existing (brownfield)
2. Create conductor/ directory structure
3. Gather project context interactively
4. Select code style guides
5. Configure workflow preferences
6. Generate initial track

## Process

### Step 1: Check existing setup

```bash
if [ -d "conductor" ]; then
  echo "Conductor already initialized"
  # Ask: "Reinitialize? (Existing data preserved)"
fi
```

### Step 2: Detect project type

```bash
if [ -d ".git" ]; then
  # Check if repo has files beyond .git
  if [ $(git ls-files | wc -l) -gt 0 ]; then
    TYPE="brownfield"
  else
    TYPE="greenfield"
  fi
else
  TYPE="greenfield"
  # Offer: "Initialize git repo? (Recommended)"
fi
```

### Step 3: Create structure

```bash
mkdir -p conductor/{styleguides,tracks}
echo '{"step": "initialized"}' > conductor/setup_state.json
```

### Step 4: Product Context (Interactive)

**Ask max 5 questions:**

1. "What are you building? (1-2 sentences)"
   → Save to `conductor/product.md`

2. "Who are your users?"
   - A) Developers
   - B) General consumers
   - C) Enterprise users
   - D) Other (specify)

3. "Main technology language?"
   - Python / JavaScript / TypeScript / Go / Rust / Other

4. "Key frameworks?" (if applicable)
   - React / Next.js / Django / FastAPI / Express / Other

5. "What's your main goal with this project?"
   → Add to product.md

**Generate:** `conductor/product.md`

### Step 5: Tech Stack

Based on answers above + brownfield detection:

**If greenfield:**
- Use stated preferences
- Ask for database choice if applicable

**If brownfield:**
```bash
# Read package.json or requirements.txt
cat package.json | grep -A 20 '"dependencies"'
cat requirements.txt
```

**Generate:** `conductor/tech-stack.md`

### Step 6: Code Style Guides

Based on detected/stated languages:

"I recommend these style guides:
- Python: PEP 8, type hints, docstrings
- TypeScript: ES6+, strict mode

Proceed with these?"

**Copy from templates:**
```bash
cp ${CLAUDE_PLUGIN_ROOT}/templates/code_styleguides/python.md conductor/styleguides/
cp ${CLAUDE_PLUGIN_ROOT}/templates/code_styleguides/typescript.md conductor/styleguides/
```

### Step 7: Workflow Configuration

"Development workflow preferences:

1. Use TDD (Test-Driven Development)?
   - [x] Yes (recommended)
   - [ ] No

2. Commit cadence:
   - [x] After each task
   - [ ] After each phase

3. Test coverage threshold: 80%"

**Generate:** `conductor/workflow.md` from template + customizations

### Step 8: Initialize Tracks

**Generate:** `conductor/tracks.md`
```markdown
# Tracks Registry

## Active Track
None

## All Tracks
<!-- Tracks appear here -->
```

### Step 9: Generate Initial Track

Ask user for the first feature/track they want to build. Then:
- Generate track ID
- Create track directory with spec.md and plan.md
- Update tracks.md

### Step 10: Commit setup (if git exists)

```bash
git add conductor/
git commit -m "conductor(setup): Initialize Conductor

- Add project context
- Add tech stack definition
- Add code style guides
- Configure workflow
- Create initial track"
```

**Save state:**
```json
{"step": "complete", "completed_at": "2024-12-30T15:30:00Z"}
```

## Resumability

If interrupted:
```bash
cat conductor/setup_state.json
# Resume from last step
```

## Final Message

```
✅ Conductor initialized!

Next steps:
- Review conductor/product.md
- Run /conductor:track to create more tracks
- Run /conductor:implement to start implementing

Tip: Commit conductor/ to share context with your team.
```
