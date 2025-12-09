<!-- SYMPHONY:START -->
# Symphony Code Conventions

**This project uses Symphony MCP for automated code convention management.**

## Critical Requirements

### 1. Before Writing Code

**Check MCP Status**: Verify Symphony MCP server is active. If unavailable, warn the user and do not proceed.

**Query Conventions**: Use `mcp__symphony__query_conventions` to retrieve relevant rules.
- Select appropriate category: security, style, documentation, error_handling, architecture, performance, testing
- Filter by languages as needed

### 2. After Writing Code

**Validate Changes**: Always run `mcp__symphony__validate_code` to check all changes against project conventions.

**Fix Violations**: Address any issues found before committing.

## Workflow

1. Verify Symphony MCP is active
2. Query conventions for your task
3. Write code
4. Validate with Symphony
5. Fix violations
6. Commit
<!-- SYMPHONY:END -->
