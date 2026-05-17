# Releasing Scheduled Actions MCP

This document describes the process of building, verifying, and releasing the Scheduled Actions MCP server.

## 🚀 Pre-Release Checklist

Before every release, you **must** run the production readiness scan:
1.  **Concurrency Check**: Run `go test -race ./...` to detect potential deadlocks or race conditions.
2.  **Security Audit**: Verify that all new Admin endpoints have `EchoRequireRole("admin")` and CSRF protection.
3.  **Migration Integrity**: Ensure all new migrations in `/migrations` have been tested against a clean database.
4.  **Frontend Polish**: Run `npm run lint` and `npm run build` to ensure no console logs or debug code leaks into production.

## 1. Build Multi-Platform Binaries

We use standard Go build commands. Ensure you are using Go 1.25+.

```bash
# Create binary directory
mkdir -p dist/bin

# Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/bin/schedule-mcp-linux-amd64 ./cmd/server
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dist/bin/schedule-mcp-linux-arm64 ./cmd/server

# macOS
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/bin/schedule-mcp-darwin-amd64 ./cmd/server
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/bin/schedule-mcp-darwin-arm64 ./cmd/server

# Windows
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/bin/schedule-mcp-windows-amd64.exe ./cmd/server
```
*Note: `-ldflags="-s -w"` reduces binary size by stripping debug symbols.*

## 2. GitHub Release & Tagging

1.  **Tagging**:
    ```bash
    git tag -a v1.1.0 -m "Release v1.1.0: Decision Nodes and Auto-Pruning"
    git push origin v1.1.0
    ```
2.  **Upload Artifacts**: Attach the binaries from `dist/bin/` to the GitHub release. The global installers depend on these exact filenames.

## 3. NPM Wrapper Release

The NPM package facilitates the installation of the pre-built Go binaries.

1.  **Sync Versions**: Update `version` in both `dist/npm/package.json` and `frontend/package.json`.
2.  **Publish**:
    ```bash
    cd dist/npm
    npm publish --access public
    ```

## 4. Post-Deployment Verification

After the Docker image is deployed, verify the system health:
- Check `/metrics` for Prometheus data.
- Verify worker registration in the **Node Registry**.
- Run a test task to confirm the **SSE Bridge** is alive.
