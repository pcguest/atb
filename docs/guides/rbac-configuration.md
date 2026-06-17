# RBAC Configuration Guide

This guide provides practical steps to configure Role-Based Access Control (RBAC) for `custosd` and `atb view` using shared secret tokens or OIDC/JWT.

## Custos Daemon (`custosd`) RBAC Configuration

`custosd` supports authentication via a shared secret bearer token or OIDC/JWT.

### Shared Secret Token

To enable authentication using a shared secret token, set the `CUSTOS_AUTH_TOKEN` environment variable when starting `custosd`. This token grants `admin` privileges. Generate it outside the repo and store it in your shell, process manager, or secret manager; do not commit it to source control.

```bash
export CUSTOS_AUTH_TOKEN="<random-operator-token>"
custosd --host 0.0.0.0 --port 9090
```

Clients can then authenticate by including an `Authorization: Bearer <random-operator-token>` header in their requests.

### OIDC/JWT Authentication

To enable OIDC/JWT authentication, configure the OIDC issuer and audience when starting `custosd`.

```bash
custosd \
  --host 0.0.0.0 \
  --port 9090 \
  --oidc-issuer "https://accounts.google.com" \
  --oidc-audience "your-client-id.apps.googleusercontent.com" \
  --default-role "viewer" # Default role for authenticated users without explicit role claims
```

`custosd` will fetch the JWKS from the OIDC issuer's `.well-known/jwks.json` endpoint and validate incoming JWTs. Roles are extracted from the `role` or `roles` claims in the JWT. If no valid role claim is found, the `--default-role` is applied.

Clients should obtain a JWT from the configured OIDC provider and include it in the `Authorization: Bearer <JWT>` header.

### Endpoint Permissions

`custosd` enforces the following role requirements for its endpoints:

| Endpoint                               | Method | Required Role      |
| :------------------------------------- | :----- | :----------------- |
| `/ingest`                              | `POST` | `operator` or `admin` |
| `/receipts`                            | `GET`  | `viewer` or higher |
| `/receipts/by-hash`                    | `GET`  | `viewer` or higher |
| `/receipts/:id`                        | `GET`  | `viewer` or higher |
| `/receipts/:id/verify`                 | `GET`  | `viewer` or higher |
| `/receipts/:id/attestation`            | `GET`  | `viewer` or higher |
| `/health`                              | `GET`  | Publicly accessible |
| `/custody/key`                         | `GET`  | Publicly accessible |

## ATB Viewer (`atb view`) RBAC Configuration

The `atb view` command also supports authentication via a session token or OIDC/JWT for its API routes (`/api/v1/*`).

### Session Token

By default, `atb view` generates a random session token at startup. This token is printed to the console and can be used to authenticate requests to the viewer's API. This token grants `admin` privileges.

```bash
atb view --bundle my-bundle.atb
# Output will include: "Serving ... at http://127.0.0.1:8080/view/#session=your-session-token"
```

You can also specify a session token manually:

```bash
atb view --bundle my-bundle.atb --session-token "<random-viewer-token>"
```

Clients can authenticate by including an `X-ATB-Session-Token: <random-viewer-token>` header or `session_token=<random-viewer-token>` query parameter.

### OIDC/JWT Authentication

To enable OIDC/JWT authentication for `atb view`, provide the OIDC issuer and audience:

```bash
atb view --bundle my-bundle.atb \
  --oidc-issuer "https://accounts.google.com" \
  --oidc-audience "your-client-id.apps.googleusercontent.com"
```

Roles are extracted from the JWT claims (`role` or `roles`). If no valid role claim is found, the default role (`viewer`) is applied.

Clients should obtain a JWT from the configured OIDC provider and include it in the `Authorization: Bearer <JWT>` header.

### Endpoint Permissions

`atb view` enforces the following role requirements for its API routes:

| Endpoint                               | Method | Required Role      |
| :------------------------------------- | :----- | :----------------- |
| `/api/v1/verification`                 | `GET`  | `viewer` or higher |
| `/api/v1/bundle/meta`                  | `GET`  | `viewer` or higher |
| `/api/v1/bundle/events`                | `GET`  | `viewer` or higher |
| `/api/v1/bundle/graph`                 | `GET`  | `viewer` or higher |
| `/api/v1/bundle/profile`               | `GET`  | `viewer` or higher |
| `/api/v1/bundle/verify/report`         | `GET`  | `viewer` or higher |
| `/api/v1/sessions`                     | `GET`  | `viewer` or higher |
| `/api/v1/sessions/by-actor`            | `GET`  | `viewer` or higher |
| `/api/v1/schema/status`                | `GET`  | `viewer` or higher |
| `/api/v1/bundle/verify`                | `POST` | `operator` or `admin` |
| `/api/v1/privacy/reveal`               | `POST` | `operator` or `admin` |
