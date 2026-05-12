# Schedule MCP Platform Maturity Plan (V4)

## 1. Hybrid Execution (Native Actions)
**Goal:** Reduce latency and cost by offloading simple logic from LLMs to native Go/JS execution.
*   **Standard Blocks:** Pre-built Go functions for HTTP, Slack, Discord, and Postgres.
*   **Code Blocks:** Integrated JavaScript sandbox (using a library like `go-otto` or `v8go`) to allow users to write custom server-side logic.
*   **Approach:** New task type `native_action`. The scheduler will route these to a specialized `NativeExecutor` instead of the MCP Sampling loop.

## 2. Advanced Execution Tracing
**Goal:** 100% transparency into every millisecond of a task's lifecycle.
*   **Dynamic Tracing Levels:**
    *   `BASIC`: Metadata only.
    *   `ON_FAILURE`: Capture full trace (resolved prompt, secrets masked, raw response) only if execution fails.
    *   `DEBUG`: Full trace captured for a set number of runs or time period.
    *   `FULL`: Persistent deep tracing for high-compliance tasks.
*   **Observability UI:** A new "Trace Timeline" component in the frontend to visualize the execution flow.

## 3. Insights & Analytics (Admin BI)
**Goal:** Move from "Log Monitoring" to "Business Intelligence".
*   **Key Metrics:** Latency distribution (P50/P99), success rate trends, token usage estimates, and workspace growth metrics.
*   **Implementation:** Expand `metrics.go` to expose aggregate Prometheus/Grafana-ready data and a custom JSON API for the Admin Dashboard.

## 4. Monetized Workflow Marketplace
**Goal:** Build a community ecosystem with clear paths for monetization.
*   **Distribution Models:**
    *   **Clone & Own:** Fork a template for independent modification.
    *   **Managed Subscription:** Link to a global template for automatic security/feature updates.
*   **Premium Tiers:** Integration with the existing Stripe backend. Premium templates will require an active "Pro/Plus" subscription or a one-time template purchase.
*   **Marketplace UI:** A grid-based discovery portal with categories (Social, DevTools, Finance), previews, and "Install" buttons.

## 5. Architectural Cleanliness & Safety
*   **Tracing Masking:** Ensure the `PromptResolver` masks secrets *before* persisting traces to the DB.
*   **Worker Heartbeats:** Enhance the "Monitor" to show specific worker node health and resource utilization.
*   **Lint/Build Hardening:** Continuous CI/CD verification to ensure no regressions as we move to a hybrid execution model.
