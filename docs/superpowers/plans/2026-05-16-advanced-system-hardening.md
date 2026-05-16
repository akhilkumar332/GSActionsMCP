# Advanced System Hardening & Feature Expansion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Elevate the platform to an advanced, industry-unique level by implementing distributed tracing, secure execution isolation, an agentic workflow engine, and a visual execution debugger.

**Architecture:**
1. **Foundation (Phase 1)**: OpenTelemetry for end-to-end observability and WASM for secure action isolation.
2. **Intelligence (Phase 2)**: State persistence and complex looping/branching for agentic workflows.
3. **Experience (Phase 3)**: A visual "Time-Travel" debugger on the Workflow Canvas.

---

## Phase 1: Advanced Foundation (Observability & Isolation)

### Task 1: OpenTelemetry Instrumentation
**Files:**
- Modify: `schedule-mcp/cmd/server/main.go`
- Modify: `schedule-mcp/cmd/server/middleware.go`
- Modify: `schedule-mcp/cmd/server/globals.go`

- [ ] **Step 1: Initialize OTel SDK**
Add OTel setup logic to `main.go` to export traces via OTLP.

- [ ] **Step 2: Instrument Echo & DB**
Apply `otelecho` middleware and `otelpgx` tracer to the connection pool.

- [ ] **Step 3: Trace Redis Pub/Sub**
Inject trace context into Redis messages to ensure traces span across worker nodes.

### Task 2: Secure Isolation with WASM (or Goja)
**Files:**
- Modify: `schedule-mcp/cmd/server/executor.go`

- [ ] **Step 1: Replace `otto` with `goja`**
`goja` provides better ES6 support and performance than `otto`. For full production isolation, we will also add a resource limit wrapper.

- [ ] **Step 2: Implement execution timeouts and memory limits**
Ensure a single script cannot consume more than 50MB of RAM or run for more than 5s.

---

## Phase 2: Agentic Workflow Engine (Intelligence)

### Task 3: Workflow State Persistence
**Files:**
- Modify: `schedule-mcp/schema.sql`
- Modify: `schedule-mcp/db/queries.sql`
- Modify: `schedule-mcp/cmd/server/scheduler.go`

- [ ] **Step 1: Add `workflow_state` table**
Store key-value pairs associated with a `task_id` or a specific execution chain.

- [ ] **Step 2: Update `handleDispatchTask` to load/save state**
Tasks can now read from `{{state.variable_name}}` and write back to state.

### Task 4: Advanced Loops & Conditional Branches
**Files:**
- Modify: `schedule-mcp/cmd/server/scheduler.go`

- [ ] **Step 1: Implement "Loop Until" logic**
Add a `loop_condition` field to the task schema.

---

## Phase 3: Visual Execution Debugger (Experience)

### Task 5: Trace Data API
**Files:**
- Modify: `schedule-mcp/cmd/server/api_handlers.go`

- [x] **Step 1: Create `GET /api/v1/tasks/:id/traces/:execution_id`**
Expose OTel trace data in a format suitable for the frontend.

### Task 6: Visual Playback on Canvas
**Files:**
- Modify: `schedule-mcp/frontend/src/pages/WorkflowCanvas.jsx`

- [x] **Step 1: Add Execution Timeline slider**
- [x] **Step 2: Highlight nodes/edges based on trace timing**

---

## Verification & Finalization

- [ ] **Phase 1 Verification**: Verify traces appearing in Jaeger/OTel backend.
- [ ] **Phase 2 Verification**: Run a 5-step loop task and verify state persistence.
- [ ] **Phase 3 Verification**: Playback a full execution on the canvas.
- [ ] **Final Build & Lint**: `go build ./cmd/server` and `npm run build`.
