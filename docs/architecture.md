# Architecture

## nabu task

```mermaid
sequenceDiagram
  autonumber
  actor User
  participant CLI as nabu CLI
  participant Cloudflared as cloudflared
  participant Access as Cloudflare Access
  participant Worker as Worker (nabu)
  participant D1 as Cloudflare D1

  User->>CLI: Runs a nabu task command
  CLI->>Cloudflared: Asks for the Access token of the Worker [exec]
  Cloudflared-->>CLI: Token, or not signed in
  CLI->>Access: Sends the request with the token [HTTPS/JSON]
  Access->>Access: Verifies the token
  Access->>Worker: Forwards the request [JWT]
  Worker->>Worker: Verifies the JWT (signature, iss, aud, exp)
  Worker->>Worker: Validates the task (shape, line limit, reason to drop)
  Worker->>D1: Writes the task and its event in one batch [binding/SQL]
  D1-->>Worker: Rows
  Worker-->>CLI: Task [JSON]
  CLI-->>User: Result [JSON or markdown]
```

## nabu task schema

```mermaid
sequenceDiagram
  autonumber
  actor User
  participant CLI as nabu CLI
  participant Cloudflared as cloudflared
  participant Access as Cloudflare Access
  participant Worker as Worker (nabu)

  User->>CLI: Runs nabu task schema
  CLI->>Cloudflared: Asks for the Access token of the Worker [exec]
  Cloudflared-->>CLI: Token, or not signed in
  CLI->>Access: Requests the task contract with the token [HTTPS]
  Access->>Access: Verifies the token
  Access->>Worker: Forwards the request [JWT]
  Worker->>Worker: Verifies the JWT (signature, iss, aud, exp)
  Worker-->>CLI: Task contract [JSON Schema]
  CLI-->>User: Task contract [JSON]
```

## Viewer

```mermaid
sequenceDiagram
  autonumber
  actor User
  participant Browser
  participant Access as Cloudflare Access
  participant Worker as Worker (nabu)
  participant D1 as Cloudflare D1

  User->>Browser: Opens the viewer
  Browser->>Access: Requests the page and the tasks [HTTPS]
  Access->>Access: Verifies the session
  Access->>Worker: Forwards the request [JWT]
  Worker->>Worker: Verifies the JWT (signature, iss, aud, exp)
  Worker->>D1: Reads every task [binding/SQL]
  D1-->>Worker: Rows
  Worker-->>Browser: Page [HTML] and tasks [JSON]
```

## Preview

```mermaid
sequenceDiagram
  autonumber
  actor User
  participant Just as just preview
  participant CF as Cloudflare API

  User->>Just: Runs [just]
  Just->>CF: Uploads a version as preview-nabu [HTTPS + wrangler login]
  CF->>CF: Verifies the wrangler login can edit Workers
  CF-->>Just: https://preview-nabu.gitt510.workers.dev
```

## Sources

| File | Root of |
|---|---|
| `web/src/schema.ts` | Task contract |
| `web/src/index.ts` | Worker and API |
| `web/build.mjs` | Viewer page |
| `main.go` | nabu CLI |

- `web/src/schema.ts` is the only file that defines a rule a task must meet
- The Worker serves the contract at `/api/schema` and refuses every write that breaks it
- `web/build.mjs` reads the contract from `web/src/schema.ts` when it builds the page
