# Releasing Scheduled Actions MCP

This document describes the process of building and releasing the Scheduled Actions MCP server.

## 1. Build Go Binaries

Binaries must be built for all supported platforms and architectures. We use standard Go build commands.

```bash
# Create a temporary directory for binaries
mkdir -p dist/bin

# Build for Linux (AMD64)
GOOS=linux GOARCH=amd64 go build -o dist/bin/schedule-mcp-linux-amd64 ./cmd/server

# Build for Linux (ARM64)
GOOS=linux GOARCH=arm64 go build -o dist/bin/schedule-mcp-linux-arm64 ./cmd/server

# Build for macOS (AMD64)
GOOS=darwin GOARCH=amd64 go build -o dist/bin/schedule-mcp-darwin-amd64 ./cmd/server

# Build for macOS (ARM64)
GOOS=darwin GOARCH=arm64 go build -o dist/bin/schedule-mcp-darwin-arm64 ./cmd/server

# Build for Windows (AMD64)
GOOS=windows GOARCH=amd64 go build -o dist/bin/schedule-mcp-windows-amd64.exe ./cmd/server
```

> **Note:** Both `dist/install.sh` and `dist/npm/install.js` now expect `amd64` naming for x86_64 architectures.

## 2. GitHub Release

1.  **Push Changes**: Ensure all changes are committed and pushed to the `main` branch.
2.  **Tag the Release**: Create a new git tag for the version.
    ```bash
    git tag -a v1.0.0 -m "Release v1.0.0"
    git push origin v1.0.0
    ```
3.  **Create Release**: Go to the GitHub repository's "Releases" page and create a new release from the tag.
4.  **Upload Artifacts**: Attach all the binaries generated in the `dist/bin/` folder to the release. The installers rely on these files being available under the "latest" release or the specific version tag.

## 3. NPM Package Release

The NPM package is a wrapper that facilitates the installation of the Go binary.

1.  **Update Version**: Update the version number in `dist/npm/package.json`.
    ```json
    "version": "1.0.0"
    ```
2.  **Optional Frontend Sync**: It is recommended to also update the version in `frontend/package.json` to maintain consistency across the project.
3.  **Publish to NPM**:
    ```bash
    cd dist/npm
    npm publish --access public
    ```

## 4. Post-Release Verification

After publishing, verify the installation:

```bash
# Test NPM installation
npm install -g @gsactions/mcp

# Verify the binary runs
schedule-mcp --help
```
