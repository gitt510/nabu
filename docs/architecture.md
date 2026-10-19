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

## Task contract

```mermaid
flowchart LR
  schema["web/src/schema.ts"]

  subgraph worker_box["Worker (nabu)"]
    api["API"]
    page["Viewer page"]
  end

  schema -- "Bundled [wrangler deploy]" --> api
  schema -- "Imported [build.mjs]" --> page
  api -- "Task contract [JSON Schema]" --> cli["nabu CLI"]
  api -- "422 and the reason [JSON]" --> cli
  page -- "Page [HTML]" --> browser["Browser"]
```

- `web/src/schema.ts` defines every rule a task must meet
- The API refuses every write that breaks the task contract
- The API serves the task contract at `/api/schema`
- The viewer page takes the task contract from `web/src/schema.ts` at build
