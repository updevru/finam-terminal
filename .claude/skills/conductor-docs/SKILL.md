---
name: conductor-docs
description: Fetches up-to-date library and framework documentation using Context7 MCP before writing code. MANDATORY for all external dependencies.
---

# Conductor Documentation Skill

## When I activate
- Before using ANY external library, framework, or API
- User asks "how do I use X library"
- Writing code that depends on third-party packages
- Answering questions about library usage

## MANDATORY Process

### 1. Identify what needs documentation
Examples:
- "react" for React
- "fastapi" for FastAPI
- "express" for Express.js
- "@types/node" for Node.js types

### 2. Resolve library ID

Use Context7 MCP to resolve the library:
```
Call: mcp__context7__resolve-library-id
  libraryName: "react"
  query: "using react hooks"
```

Returns: `/facebook/react`

### 3. Fetch documentation

Use Context7 MCP to get docs:
```
Call: mcp__context7__query-docs
  libraryId: "/facebook/react"
  query: "how to use useEffect hook with cleanup"
```

### 4. Use current docs
- Extract relevant API information
- Use current patterns (not training data)
- Cite docs in code comments if helpful

## When to SKIP Context7

**Standard library (no Context7 needed):**
- Python: `os`, `sys`, `json`, `datetime`
- Node.js: `fs`, `path`, `http`
- Basic language features: loops, conditionals, classes

**Project's own code:**
- Internal modules
- Custom utilities
- Local files

## If Context7 fails

1. **Ask user**: "Is 'library-name' correct? Try alternative name?"
2. **Check if it's standard library** (skip Context7)
3. **Fall back to training knowledge** + warn user:
   ```
   ⚠️ Couldn't fetch docs for X. Using training data (may be outdated).
   Please verify against official docs.
   ```

## Examples

**Good:**
```
User: "Add authentication with JWT"
→ resolve-library-id("jsonwebtoken")
→ query-docs("/auth0/jsonwebtoken", "how to sign and verify tokens")
→ Use current JWT API
```

**Skip:**
```
User: "Read a JSON file"
→ This is Python stdlib / Node.js built-in
→ No Context7 needed
```

## Important

- Always call Context7 for external deps
- Keep only essential API snippets in memory
- Don't reproduce entire docs
