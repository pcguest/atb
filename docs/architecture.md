# ATB architecture

## Component overview

```mermaid
flowchart LR
    CLI["CLI<br/>(atb)"] --> Core["Core Engine"]
    MCP["MCP Server<br/>(atb mcp serve)"] --> Core
    PythonSDK["Python SDK"] --> Core
    TypeScriptSDK["TypeScript SDK"] --> Core
    Core --> BundleStore["Bundle Store<br/>(.atb file)"]
    BundleStore --> Verify["Verify<br/>(hash chain + profiles)"]
    BundleStore --> Export["Export<br/>(compliance / soc2 / gdpr)"]
    BundleStore --> Push["Push — v1.6<br/>(S3 / WORM)"]
    Verify --> Report["Trust Report"]
    Export --> Archive["Evidence Archive"]
    Push --> RemoteStorage["Remote WORM Storage"]
```

## Trust boundary

The trust boundary runs at the bundle file boundary. The Core Engine writes and reads the bundle; everything above the boundary (CLI commands, MCP tool calls, SDK calls) must go through the Core Engine's append and verify logic. No component reads or writes bundle records directly.

Integrity is verified at the file boundary on read: `atb verify` runs the hash chain across all records before returning results. The dashboard (`atb view`) runs the same check before serving event data — if verification fails, the data endpoints return `403`.

Export and push operations seal the bundle before writing. A bundle that fails verification cannot be exported or pushed; the operator must address the integrity issue first.

The `Push — v1.6` path is planned and not yet implemented. See [`docs/integrations/worm-s3.md`](./integrations/worm-s3.md) for the design intent.
