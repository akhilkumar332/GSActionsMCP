# Security Policy

## Supported Versions

We provide security updates for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | ✅ Yes             |
| < 1.0   | ❌ No              |

## Security Architecture

Schedule MCP implements a **Multi-Layer Defense** strategy:

### 1. Data Integrity & Privacy
*   **Zero-Trust Secret Vault**: All task secrets and API keys are encrypted with **AES-256-GCM**. Decryption only occurs in-memory during task execution.
*   **Tracing Anonymization**: Internal tracing (`execution_traces`) stores raw data as TEXT to prevent injection vulnerabilities associated with dynamic JSON parsing.

### 2. Authentication & Access Control
*   **Database-Backed Sessions**: Allows for immediate global session revocation.
*   **Granular RBAC**: Strict Role-Based Access Control (Admin/Staff/User) enforced at the middleware layer.
*   **Self-Demotion Block**: Admins are restricted from removing their own privileges to prevent system lockouts.

### 3. Network & API Security
*   **Hardened CSRF**: Strict origin validation and double-submit token enforcement on all mutation endpoints.
*   **Quota Enforcement**: Centralized quota logic ensures API users cannot bypass tier-based task limits.
*   **SSE Isolation**: Every persistent bridge connection is cryptographically linked to a user session.

### 4. Secure Execution
*   **Native Sandboxing**: Custom JS actions run in an isolated environment (Goja) with strict CPU timeouts and memory caps.

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability, please **do not open a public issue**. Instead, follow the process below:

1.  **Draft a Report**: Include a detailed description of the vulnerability, steps to reproduce, and potential impact.
2.  **Submit Privately**: Send your report via email to `akhilkumar332@gmail.com`.
3.  **Wait for Response**: We will acknowledge your report within 48 hours and provide a timeline for a fix.

Thank you for helping keep the future of AI orchestration secure.
