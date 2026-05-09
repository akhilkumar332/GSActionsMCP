# React Frontend Migration & Landing Page Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Transform the `schedule-mcp` management portal into a modern React.js SPA with a high-fidelity landing page and a robust JSON API backend.

**Architecture:** We will decouple the frontend from the Go backend. The Go server will serve a RESTful JSON API under `/api/*` and host the static React assets. We'll use Vite for the frontend build and TailwindCSS for a modern "Anthropic-inspired" aesthetic.

**Tech Stack:** React 19, Vite, TailwindCSS, Lucide Icons, Go 1.25, gorilla/csrf (modified for API).

---

### Task 1: Backend API Refactoring

**Files:**
- Modify: `schedule-mcp/cmd/server/main.go`
- Modify: `schedule-mcp/cmd/server/middleware.go`
- Create: `schedule-mcp/cmd/server/api_handlers.go`

- [ ] **Step 1: Create JSON Response Helper**
Implement a `sendJSON` helper in `api_handlers.go` to standardize API responses.

- [ ] **Step 2: Convert Auth Handlers to JSON**
Move `/signup`, `/login`, `/logout` logic to JSON-only endpoints under `/api/auth/`.

- [ ] **Step 3: Convert Dashboard & Admin Handlers to JSON**
Implement `/api/dashboard`, `/api/rotate-api-key`, `/api/monitor`, and `/api/admin/users` as JSON endpoints.

- [ ] **Step 4: Update Middleware for API**
Ensure `sessionMiddleware` and `RequireRole` return JSON errors (401/403) instead of redirects for `/api/*` routes.

- [ ] **Step 5: Verify API with curl**
Run: `go build ./cmd/server` and test `/api/healthz` and other endpoints.

---

### Task 2: Frontend Project Setup (Vite + Tailwind)

**Files:**
- Create: `schedule-mcp/frontend/` (Vite project)

- [ ] **Step 1: Initialize Vite project**
Run: `npm create vite@latest frontend -- --template react`

- [ ] **Step 2: Install dependencies**
Run: `cd frontend && npm install tailwindcss @tailwindcss/vite lucide-react axios react-router-dom`

- [ ] **Step 3: Configure Tailwind**
Add Tailwind to `vite.config.js` and setup `index.css` with the "Paper & Ink" theme.

- [ ] **Step 4: Setup API Proxy**
Configure Vite to proxy `/api` requests to `http://localhost:8080` for development.

---

### Task 3: Modern Landing Page & Global Components

**Files:**
- Create: `frontend/src/pages/Landing.jsx`
- Create: `frontend/src/components/Hero.jsx`

- [ ] **Step 1: Build the Hero Section**
Create a stunning hero with glassmorphism, bold Poppins headers, and a "Get Started" CTA.

- [ ] **Step 2: Implement "Detailed Tool Showcase"**
Add sections for Persistent Scheduling, Reliable Sampling, and RBAC.

- [ ] **Step 3: Installation & Steps Section**
Interactive step-by-step guide on connecting to Claude Desktop or Cursor.

---

### Task 4: React Dashboard & RBAC Migration

**Files:**
- Create: `frontend/src/pages/Dashboard.jsx`
- Create: `frontend/src/pages/Monitor.jsx`
- Create: `frontend/src/context/AuthContext.jsx`

- [ ] **Step 1: Implement AuthContext**
Handle session state, login, and logout globally in the React app.

- [ ] **Step 2: Build the Bento Dashboard**
Re-create the dashboard in React using Tailwind grids. Include API key rotation with immediate UI feedback.

- [ ] **Step 3: Build Staff Monitor & Admin Views**
Implement the terminal-style log viewer and the user management table with real-time updates.

---

### Task 5: Integration & Production Build

**Files:**
- Modify: `schedule-mcp/cmd/server/main.go`
- Modify: `schedule-mcp/Dockerfile`

- [ ] **Step 1: Configure Go to serve React SPA**
Add a catch-all handler in `main.go` that serves the built React app.

- [ ] **Step 2: Update Dockerfile for Multi-stage Build**
Build frontend with Node, then copy to Go stage for a single production image.

- [ ] **Step 3: Final Verification**
Run complete system and verify all flows: Sign Up -> Login -> Dashboard -> Task Scheduling.
