---
name: conductor-workflow
description: Enforces Conductor's TDD task lifecycle and plan management. Use during /conductor:implement to ensure proper task execution, testing, commits, and plan updates.
---

# Conductor Workflow Skill

## When I activate
- User runs `/conductor:implement`
- Implementing any task from a plan
- Need to know how to execute tasks properly
- Updating plan status

## Task Lifecycle (Strict Order)

### Phase 1: Pre-work
1. Check git status (warn if dirty)
2. Find next pending task: first `[ ]` in plan.md
3. Mark task as `[~]` in plan.md (in-progress)
4. Read task requirements and acceptance criteria

### Phase 2: TDD Cycle (if workflow.md says TDD)
1. **Write test first**
   - Create or update test file
   - Test should FAIL initially

2. **Run test and verify failure**
   ```bash
   npm test  # or appropriate test command
   ```
   - If test passes initially → test is wrong, fix it

3. **Implement code**
   - Write minimal code to make test pass
   - Follow style guides from conductor/styleguides/

4. **Run test and verify pass**
   - All tests must pass
   - If fails → fix implementation

5. **Refactor if needed**
   - Clean up code
   - Tests still pass

### Phase 3: Commit
1. **Stage changes**
   ```bash
   git add <modified files>
   ```

2. **Commit with metadata**
   ```bash
   git commit -m "feat(track_<id>): Task description

   - Detailed changes
   - What was implemented

   Track: track_<id>
   Phase: <phase_number>
   Task: <task_number>"
   ```

3. **Get commit SHA**
   ```bash
   git rev-parse --short HEAD
   ```

### Phase 4: Update Plan
1. **Mark task complete**
   - Change `[~]` to `[x]` in plan.md
   - Append commit SHA: `[x] Task description (abc1234)`

2. **Commit plan update**
   ```bash
   git commit -am "docs(track_<id>): Update plan status"
   ```

### Phase 5: Verification
1. **If task is last in phase:**
   - Run full test suite
   - Ask user: "Phase complete. Please verify manually. Continue to next phase?"
   - Wait for explicit approval

2. **If track complete:**
   - Ask user: "Track complete! Mark as done in tracks.md?"

## Non-TDD Workflow (Alternative)

If workflow.md says "no TDD":
1. Implement code first
2. Write tests after
3. Run tests
4. Commit (same format)
5. Update plan

## Error Handling

**Test failures:**
- Show error output
- Ask user: "Fix implementation? Skip task? Abort?"

**Git failures:**
- Show git error
- Ask user: "Resolve conflict? Abort?"

**Missing files:**
- If workflow.md missing → ask user to run /conductor:setup
- If plan.md missing → ask user to run /conductor:track

## Important Rules

- ✅ ONE task at a time (never skip ahead)
- ✅ MUST mark `[~]` before starting
- ✅ MUST commit after each task
- ✅ MUST get user approval between phases
- ❌ NEVER auto-proceed to next phase
- ❌ NEVER skip tests
