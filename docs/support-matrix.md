# Support matrix

This file is the single maintainer-facing support matrix for ATB builds,
tests, SDKs, and the local viewer. The CI version gate checks these values
against repository configuration.

| Area | Supported or tested version | Source enforced by CI |
| --- | --- | --- |
| Go toolchain | Go 1.26.5 | `go.mod`, `Makefile`, GitHub Actions `setup-go` |
| Python SDK runtime | Python 3.9-3.12 | `sdk/python/pyproject.toml` classifiers and `requires-python` |
| Python CI runtime | Python 3.11 | GitHub Actions `setup-python` |
| TypeScript SDK runtime | Node.js >=18 | `sdk/typescript/package.json` `engines.node` |
| Node.js CI runtime | Node.js 22 | GitHub Actions `setup-node` |
| Package manager | npm with committed `package-lock.json` files | GitHub Actions use `npm ci` |
| Local viewer browser | Current Chromium-family, Firefox, and Safari browsers | Viewer build and typecheck in CI; browser-specific support is operationally tested before release |

The TypeScript SDK keeps a wider runtime engine range than CI so downstream
applications on maintained Node.js LTS releases can consume the package. The
repository itself builds and tests Node-based packages on Node.js 22.

Changes to this matrix must update the corresponding workflow, manifest, and
script checks in the same PR.
