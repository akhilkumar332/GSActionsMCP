# Workspace Environment Variables Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement workspace-level environment variables that can be shared across all tasks within a workspace and injected into prompts using `{{env.KEY}}`.

**Architecture:**
1. **Storage:** A new `workspace_env_vars` table linked to workspaces.
2. **Management:** CRUD endpoints for environment variables with workspace-level authorization.
3. **Injection:** The `prompt_resolver.go` will be updated to fetch environment variables for the task's workspace and perform regex replacement.

**Tech Stack:** Go (Echo), PostgreSQL (sqlc), React.

---

### Task 1: Database Migration for Workspace Env Vars

**Files:**
- Create: `migrations/015_workspace_env_vars.up.sql`
- Create: `migrations/015_workspace_env_vars.down.sql`
- Modify: `db/queries.sql`

- [ ] **Step 1: Write migration up script**

```sql
CREATE TABLE workspace_env_vars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    value TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (workspace_id, name)
);

CREATE INDEX idx_workspace_env_vars_workspace_id ON workspace_env_vars (workspace_id);
```

- [ ] **Step 2: Write migration down script**

```sql
DROP TABLE IF EXISTS workspace_env_vars;
```

- [ ] **Step 3: Add CRUD queries to db/queries.sql**

```sql
-- name: UpsertWorkspaceEnvVar :one
INSERT INTO workspace_env_vars (workspace_id, name, value)
VALUES ($1, $2, $3)
ON CONFLICT (workspace_id, name) DO UPDATE SET
    value = EXCLUDED.value,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: ListWorkspaceEnvVars :many
SELECT * FROM workspace_env_vars WHERE workspace_id = $1 ORDER BY name ASC;

-- name: DeleteWorkspaceEnvVar :exec
DELETE FROM workspace_env_vars WHERE workspace_id = $1 AND name = $2;

-- name: GetTaskWorkspaceEnvVars :many
SELECT e.name, e.value 
FROM workspace_env_vars e
JOIN tasks t ON e.workspace_id = t.workspace_id
WHERE t.id = $1;
```

- [ ] **Step 4: Run sqlc generate**

Run: `go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate`

- [ ] **Step 5: Commit**

```bash
git add migrations/ db/
git commit -m "feat(db): add schema for workspace environment variables"
```

---

### Task 2: Backend Logic - CRUD Handlers for Env Vars

**Files:**
- Modify: `cmd/server/workspace_handlers.go`
- Modify: `cmd/server/main.go`

- [ ] **Step 1: Implement CRUD Handlers**

Add `handleUpsertWorkspaceEnvVar`, `handleListWorkspaceEnvVars`, and `handleDeleteWorkspaceEnvVar` to `cmd/server/workspace_handlers.go`. 
Ensure strict `CheckWorkspaceAccess` is performed for each request.

- [ ] **Step 2: Register routes in main.go**

```go
v1.GET("/workspaces/:id/env", handleListWorkspaceEnvVars, EchoSessionMiddleware)
v1.POST("/workspaces/:id/env", handleUpsertWorkspaceEnvVar, EchoSessionMiddleware)
v1.DELETE("/workspaces/:id/env/:name", handleDeleteWorkspaceEnvVar, EchoSessionMiddleware)
```

- [ ] **Step 3: Commit**

```bash
git add cmd/server/
git commit -m "feat(backend): implement CRUD endpoints for workspace environment variables"
```

---

### Task 3: Prompt Injection Logic

**Files:**
- Modify: `cmd/server/prompt_resolver.go`

- [ ] **Step 1: Update resolvePrompt to handle {{env.KEY}}**

1. Create a new regex: `var envVarRegex = regexp.MustCompile(\`\{\{env\.([a-zA-Z0-9_-]+)\}\}\`)`.
2. Update the `resolvePrompt` function signature (or the internal logic) to fetch workspace env vars if the task belongs to a workspace.
3. Replace all matches of `{{env.KEY}}` with the corresponding values from `workspace_env_vars`.

- [ ] **Step 2: Commit**

```bash
git add cmd/server/prompt_resolver.go
git commit -m "feat(backend): implement {{env.KEY}} prompt injection"
```

---

### Task 4: Frontend - Workspace Env Var Management UI

**Files:**
- Modify: `frontend/src/pages/Workspaces.jsx`

- [ ] **Step 1: Enhance Workspaces page**

1. When a user clicks a workspace, show an expandable section or modal to manage Environment Variables.
2. Implement a list of existing variables.
3. Add a form to add/update a variable (Name/Value).
4. Add a delete button for each variable.
5. Use existing Tailwind/Framer styles.

- [ ] **Step 2: Commit**

```bash
git add frontend/
git commit -m "feat(frontend): implement workspace environment variable management UI"
```

---

### Task 5: Verification and Build

- [ ] **Step 1: Run Go Lint and Build**

Run: `go build ./cmd/server`
Expected: Success

- [ ] **Step 2: Run Frontend Build**

Run: `cd frontend && npm run build`
Expected: Success

- [ ] **Step 3: Commit**

```bash
git commit -m "chore: verify workspace environment variables implementation"
```
