# Architecture

- Each route below is a C4 dynamic diagram at the level of people, containers, and external systems

## Task write with auto_push

```mermaid
sequenceDiagram
  autonumber
  actor User
  participant CLI as nabu CLI
  participant Notebook as Notebook (local git repository)
  participant GitHub as GitHub (notebook repository)

  User->>CLI: Runs a write command, such as nabu task new
  CLI->>Notebook: Writes the task file and commits it [git]
  Notebook-->>CLI: Commit
  CLI->>CLI: Checks auto_push is on
  CLI->>CLI: Checks the branch has an upstream
  CLI->>GitHub: Pushes the current branch [git/HTTPS]
  GitHub-->>CLI: Push accepted
  CLI-->>User: Path, commit, and push result [JSON]
```

## nabu publish

- The nightly schedule starts the same route from step 8

```mermaid
sequenceDiagram
  autonumber
  actor User
  participant CLI as nabu CLI
  participant GitHub as GitHub (notebook repository)
  participant Actions as GitHub Actions (publish workflow)
  participant R2 as Cloudflare R2

  User->>CLI: Runs nabu publish
  CLI->>CLI: Checks the branch has an upstream
  CLI->>GitHub: Pushes the current branch [git/HTTPS]
  GitHub-->>CLI: Push accepted
  CLI->>GitHub: Dispatches publish.yml on the branch [gh/HTTPS]
  GitHub-->>CLI: Run URL
  CLI-->>User: Upstream, ref, and run URL [JSON]
  GitHub->>Actions: Starts the publish run [workflow_dispatch]
  Actions->>GitHub: Clones the notebook with full history [git/HTTPS]
  GitHub-->>Actions: Commits
  Actions->>R2: Replaces tasks.json [Wrangler/HTTPS]
  R2-->>Actions: 200 OK
```

## Viewer reads tasks

```mermaid
sequenceDiagram
  autonumber
  actor User
  participant Browser as Browser (viewer page)
  participant Access as Cloudflare Access
  participant Worker as Worker (nabu)
  participant R2 as Cloudflare R2

  User->>Browser: Opens the viewer
  Browser->>Access: Requests the page [HTTPS]
  Access->>Access: Verifies the session
  Access->>Worker: Forwards the request [HTTPS]
  Worker-->>Browser: Page [HTML/JS/CSS]
  Browser->>Access: Requests tasks.json [HTTPS]
  Access->>Worker: Forwards the request [HTTPS]
  Worker->>R2: Reads tasks.json [R2 binding]
  R2-->>Worker: tasks.json
  Worker-->>Browser: 200 OK [JSON]
```

## Viewer deploy

```mermaid
sequenceDiagram
  autonumber
  actor Developer
  participant GitHub as GitHub (gitt510/nabu)
  participant Actions as GitHub Actions (deploy workflow)
  participant Cloudflare as Cloudflare Workers

  Developer->>GitHub: Merges into main [git/HTTPS]
  GitHub->>Actions: Starts the deploy run [push]
  Actions->>GitHub: Clones nabu [git/HTTPS]
  GitHub-->>Actions: Commits
  Actions->>Actions: Runs the Go tests
  Actions->>Cloudflare: Deploys the Worker and the viewer page [Wrangler/HTTPS]
  Cloudflare-->>Actions: Deployed version
```
