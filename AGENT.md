# AGENT.md — Complete guide to the checkship codebase

> Audience: AI coding agents and developers working on checkship, a fork of seelf renamed and
> rebranded by checkship-deploy (see §11–§13).
> Everything here was derived from reading the full source tree (backend, frontend, docs, CI,
> migrations, examples) at version **2.4.2** (commit `d4703a3`).

---

## 0. TL;DR

- **What it is:** a self-hosted PaaS ("deploy your `compose.yml` to your own servers"). It's an alternative
  to Heroku, Dokku, CapRover and Coolify. It ships as a **single Go binary** that embeds a **SvelteKit SPA**,
  stores everything in **SQLite**, and drives **Docker / Docker Compose** (locally or over **SSH**).
  It also deploys a **Traefik v2.11** proxy per target so apps get subdomains and Let's Encrypt certificates automatically.
- **Architecture:** DDD + CQRS + domain events + an in-process message **bus** + a persistent
  SQLite-backed **job scheduler** (background workers).
- **Bounded contexts:** `internal/auth` (users, sessions, API key) and `internal/deployment`
  (targets, apps, deployments, registries).
- **Original author:** Julien Leicher (GitHub `YuukanOO`). **License: GPL-3.0** (see §12). checkship is a renamed fork of **seelf** v2.4.2 (see §11).
- **Go module path:** `github.com/checkship-deploy/checkship` (936 import lines reference it).

---

## 1. Repository map

```
.
├── main.go                     # entry: cmd.Root().Execute()
├── cmd/
│   ├── root.go                 # cobra root cmd "checkship", -c/--config flag, logger + config init
│   ├── version/version.go      # var version = "2.4.2" (rewritten by `make prepare-release`)
│   ├── config/configuration.go # config struct, defaults, env/yaml loading, validation
│   ├── startup/server.go       # composition root: DB, bus, scheduler, module setup, first user, expose self
│   └── serve/                  # `serve` subcommand = HTTP API (gin) + embedded SPA
│       ├── cmd.go, server.go, middlewares.go
│       ├── apps.go deployments.go targets.go registries.go jobs.go sessions.go users.go healthcheck.go
│       └── front/              # SvelteKit frontend (built into front/build, embedded via go:embed)
├── internal/
│   ├── auth/                   # bounded context: users
│   │   ├── domain/ app/ infra/ fixture/
│   └── deployment/             # bounded context: targets, apps, deployments, registries
│       ├── domain/             # entities, value objects, events, interfaces (ports)
│       ├── app/<usecase>/      # one package per command/query + event handlers (on_*.go)
│       ├── infra/              # sqlite stores, docker provider, sources, artifact manager, mod.go wiring
│       └── fixture/            # test builders (build tag !release)
├── pkg/                        # generic, reusable libs (not checkship-specific)
│   ├── bus/ (memory/, sqlite/, spy/)  # mediator bus + scheduler + job store
│   ├── storage/ (sqlite/, sqlite/builder/) # DB wrapper, migrations, tiny query builder, dataloaders
│   ├── monad/ (Maybe, Patch)   validate/ apperr/ event/ ssh/ config/ log/ id/ crypto/ ...
├── docs/                       # VitePress documentation site (deployed to GitHub Pages under /checkship/)
├── examples/                   # sample apps (minimal, go-api, sveltekit-hello, multiple-ports)
├── Dockerfile, compose.yml, Makefile, api.http, .releaserc, .github/
└── package.json                # only vitepress (docs)
```

File counts: ~477 tracked files, 77 Go test files, 6 frontend test files.

---

## 2. Tech stack

| Layer | Tech |
|---|---|
| Language | Go 1.23 (`go.mod`), CGO required (sqlite3) |
| CLI | spf13/cobra |
| HTTP | gin-gonic/gin + gin-contrib/sessions (encrypted **cookie** store) |
| DB | SQLite via mattn/go-sqlite3 (WAL, FK on, `_txlock=immediate`), golang-migrate (embedded SQL) |
| Logging | uber/zap (console/json) |
| IDs | segmentio/ksuid (27-char, time-sortable) |
| Docker | docker/cli + docker/docker client + **docker/compose v2 as a library** + compose-spec/compose-go |
| Git | go-git (pure Go clone/checkout; no git binary needed) |
| SSH | system `ssh` (used by the docker CLI `ssh://` transport) + kevinburke/ssh_config to edit `~/.ssh/config` |
| Proxy | Traefik `v2.11` (deployed as a compose project on each target) |
| Frontend | SvelteKit 1.20 (adapter-static, SPA mode, SSR off), Svelte 3, TypeScript 4.9, CSS modules, Vite 4, Vitest |
| Docs | VitePress 1.x |
| Release | semantic-release → bumps `cmd/version/version.go`, builds multi-arch Docker image → GitHub Container Registry `ghcr.io/checkship-deploy/checkship` |

`go.mod` **replace** directives (important; keep them when renaming):
```
github.com/docker/compose/v2   => github.com/YuukanOO/compose/v2 ...   # skips network aliases on external networks (DNS conflicts)
github.com/kevinburke/ssh_config => github.com/YuukanOO/ssh_config ... # exposes IsImplicit etc.
```
These are the original author's **forks of third-party libraries**. They are dependencies, not branding.
Renaming them breaks the build unless you fork those repos yourself.

---

## 3. Running, building, testing

```sh
make serve-back     # ADMIN_EMAIL=admin@example.com ADMIN_PASSWORD=admin LOG_LEVEL=debug go run main.go serve  (API on :8080)
make serve-front    # vite dev server; proxies /api/v1 -> http://127.0.0.1:8080
make serve-docs     # vitepress dev
make test           # front (vitest) + back (go vet ./... && go test ./... --cover)
make build          # build-front (npm run build -> cmd/serve/front/build) then build-back (go build -tags release -ldflags="-s -w" -o checkship)
make ts             # prints unix timestamp, used as migration filename prefix
make outdated       # list direct deps with newer versions
make prepare-release CHECKSHIP_VERSION=x.y.z   # sed version.go
make release-docker CHECKSHIP_VERSION=x.y.z    # buildx amd64+arm64 push ghcr.io/checkship-deploy/checkship
```

Gotchas:
- **Build the frontend before the backend.** `cmd/serve/server.go` has `//go:embed all:front/build/*`.
  Only `front/build/.keep` is committed, so a backend-only build serves no UI (`fallback.html` is missing).
- `-tags release` excludes `//go:build !release` files: `pkg/bus/spy`, `internal/*/fixture`.
- Windows builds need gcc (sqlite3 CGO).
- VS Code launch config: `.vscode/launch.json` → `checkship:serve` with debug env.
- `api.http` is a REST-client scratch file covering every endpoint.
- Dockerfile: `node:18-alpine` builds the front, `golang:1.23-alpine` (+build-base) builds the back, and the
  runtime is `alpine:3.16` + `openssh-client`. `DATA_PATH=/checkship/data`, CMD `./checkship -c /checkship/data/conf.yml serve`, port 8080.
- CI (`.github/workflows/ci.yml`) uses `go-version: "1.21"` while go.mod says `go 1.23`. This is a mismatch
  that only works through Go's automatic toolchain download, so bump it.

---

## 4. Configuration (`cmd/config/configuration.go`, `pkg/config/loader.go`)

Load order: **YAML file** (`-c`, default `~/.config/checkship/conf.yml`) → **`.env`, `.env.local`** (cwd) →
**environment variables** (these win). If the config file doesn't exist, it is **written** with the
current values, which persists the generated HTTP secret. Then `PostLoad()` validates the values.

| YAML | ENV | Default | Notes |
|---|---|---|---|
| log.level | LOG_LEVEL | info | debug/info/warn/error; debug enables the request logger |
| log.format | LOG_FORMAT | console | console/json |
| data.path | DATA_PATH | `os.UserConfigDir()/checkship` | DB, apps, logs live here |
| data.deployment_dir_template | DEPLOYMENT_DIR_TEMPLATE | `{{ .Environment }}` | Go template; fields `.Number`, `.Environment` |
| http.host | HTTP_HOST | "" (all) | |
| http.port | HTTP_PORT, PORT | 8080 | |
| http.secure | HTTP_SECURE | unset → derived from EXPOSED_ON scheme | cookie `Secure` flag + `Location` header scheme |
| http.secret | HTTP_SECRET | random 64 chars | cookie signing/encryption key |
| runners | RUNNERS_CONFIGURATION | deploy runner (4s, 4 workers) + catch-all (4s, 2) | list of `{poll_interval, count, jobs}`; env format `4s;4;deployment.command.deploy\|4s;2;`. Code in `cmd/config/runners.go` |
| runners.poll_interval/deployment/cleanup | RUNNERS_POLL_INTERVAL, RUNNERS_DEPLOYMENT_COUNT, RUNNERS_CLEANUP_COUNT | | deprecated, still read into the default pair of runners with a warning |
| — | CHECKSHIP_ADMIN_EMAIL, ADMIN_EMAIL | | first user only |
| — | CHECKSHIP_ADMIN_PASSWORD, ADMIN_PASSWORD | | first user only |
| — | EXPOSED_ON | | `<scheme>://<container>@<root-domain>` e.g. `http://checkship@docker.localhost` |

SQLite DSN: `file:<data.path>/checkship.db?_journal=WAL&_timeout=5000&_foreign_keys=yes&_txlock=immediate`.

Data directory layout at runtime:
```
<DATA_PATH>/
  conf.yml                 # (when -c points here, as in Docker)
  checkship.db (+ -wal, -shm)
  apps/<appID>/<template>/ # build dir; emptied before each deployment (default: apps/<id>/production)
  logs/<unixRequestedAt>-<appID>-<number>.deployment.log
~/.ssh/config, ~/.ssh/<targetID>   # remote targets' SSH entries + private keys (0600)
$TMPDIR/checkship-archive-*            # uploaded tar.gz archives (never cleaned up, see §13)
```

---

## 5. System design

### 5.1 Component diagram

```
 Browser (SvelteKit SPA)      CI / curl (Bearer API key)
          │  cookie session            │
          ▼                            ▼
 ┌──────────────────── gin HTTP server (cmd/serve) ────────────────────┐
 │ /api/v1/*  → handlers → bus.Send(Command|Query)                     │
 │ everything else → embedded SPA (front/build, fallback.html)         │
 └───────────────┬─────────────────────────────────────────────────────┘
                 ▼
     ┌──── in-memory Bus (pkg/bus/memory) ──────┐
     │  Request handlers (1 per message name)   │
     │  Signal handlers (N per event name)      │
     └──┬────────────────────────────┬──────────┘
        │ commands                   │ queries
        ▼                            ▼
  Domain entities (App, Target,   SQL "gateway" read models
  Deployment, Registry, User)     (infra/sqlite/gateway.go)
        │ raise events
        ▼
  Stores.Write → WriteAndDispatch (one SQL tx):
     event → SQL INSERT/UPDATE  +  bus.Notify(event) → signal handlers
                                       │ (same tx)
                                       ▼
                            Scheduler.Queue(job) → scheduled_jobs table
                                       │
  ┌────────────────────────────────────┴─────────────────────────────┐
  │ Scheduler (pkg/bus/scheduler.go), one poller per runner:         │
  │   • deploy runner (4 workers): deployment.command.deploy         │
  │   • catch-all runner (2 workers): cleanup_app, delete_app,       │
  │       configure_target, cleanup_target, delete_target            │
  │ woken after commit, poll every 4s as fallback                    │
  │ each job → bus.Send(msg) → Done (delete row) | Retry (+15s)      │
  └────────────────────────────────────┬─────────────────────────────┘
                                       ▼
   Source facade (raw | archive | git) → build dir
   Provider facade → Docker provider → docker API / compose lib
        │ local unix socket  or  ssh://user@host (via ~/.ssh/config)
        ▼
   TARGET HOST: Traefik proxy project "checkship-internal-<target>"
                network "checkship-gateway-<target>"
                app projects "<app>-<env>-<appid>"
```

### 5.2 Startup sequence (`cmd/startup/server.go`)
1. `memory.NewBus()`.
2. `sqlite.Open(dsn)`.
3. `bussqlite.NewScheduledJobsStore(db).Setup()` migrates the `scheduler` module and resets `retrieved=true` jobs
   to `false`, so jobs that were running during a crash are picked up again.
4. `bus.NewScheduler(store, logger, bus)`. Jobs can be queued from here on; nothing runs until step 9.
5. `authinfra.Setup` registers auth handlers and migrates `auth`.
6. `deploymentinfra.Setup` builds the stores, artifact manager, source facade (raw, archive, git) and provider facade (docker),
   registers all request + signal handlers, migrates `deployment`, then **fails all `Running` deployments**
   with errcode `server_reset`.
7. `create_first_account`: if no user exists, it needs ADMIN_EMAIL and ADMIN_PASSWORD (otherwise it errors with a help message).
8. If `EXPOSED_ON` is set, `expose_checkship_container` runs (see §6.8).
9. `options.RunnersDefinitions(bus.Marshallable.Keys(), logger)` checks the runners config against every registered job
   type (unknown or unhandled names stop startup), then `scheduler.Start(runners...)`, then the HTTP server listens. Graceful shutdown on SIGINT/SIGTERM (5s), then
   `scheduler.Stop()` waits for running jobs and the DB closes.

### 5.3 The bus (`pkg/bus`)
- Messages implement `Name_()` + `Kind_()` (underscore suffix avoids clashing with fields called `Name`).
- `bus.Command[T]` / `bus.Query[T]` are embedded in a struct to declare the typed result. `bus.Notification` is for events.
- `bus.Register(b, handler)` registers one handler per request name (it panics on duplicates). If the message implements
  `Schedulable` (`ResourceID()`), it is also registered in `bus.Marshallable` so it can be JSON-rehydrated from the job table.
- `bus.On(b, handler)` registers N handlers per signal. `Notify` calls them in **registration order** and stops at the first error.
- `bus.Send[TResult](b, ctx, msg)` is the type-safe dispatch.
- The `spy` dispatcher is for tests: it records requests and signals.

### 5.4 Persistence + events (`pkg/storage/sqlite/database.go`)
- Entities embed `event.Emitter`. Mutations call `apply(evt)`, which updates the fields and stores the event.
- `Store.Write(ctx, entities...)` → `WriteAndDispatch`: opens a tx (or reuses the tx already in the ctx). For each event it runs the
  store's `switch evt.(type)` to SQL, then calls `bus.Notify(events...)` **inside the same transaction**. Signal handlers that
  write other entities or queue jobs therefore commit or roll back atomically with the original change.
- Events are **never cleared** after writing (TODO #71). **Never call `Write` twice on the same entity instance**;
  reload it instead, as the handlers do with "fresh" re-reads.
- Rehydration: `XxxFrom(scanner)` constructors in `domain` scan the columns in a fixed order.
- Discriminated unions (`storage.DiscriminatedMapper`): `domain.SourceDataTypes` (raw/archive/git),
  `domain.ProviderConfigTypes` (docker), plus read-model twins in `get_deployment` / `get_target`. Concrete types
  register in `init()` of their infra package.
- `pkg/storage/sqlite/builder`: `Query[T]`, `Select[T]` (paginated: swaps fields for `COUNT(*)`), `Insert`, `Update`,
  `Command`, statements `Maybe`, `MaybeValue`, `If`, `Array` (IN clause), and **Dataloaders** to batch-load related data (no N+1).
- Migrations: one `MigrationsModule` per context (`scheduler`, `auth`, `deployment`), each with its own history table
  `<module>_schema_migrations`. Files are named `<unix_ts>_<name>.up.sql` (use `make ts`). **Never edit an applied migration.**

### 5.5 Scheduler / jobs (`pkg/bus/scheduler.go`, `pkg/bus/sqlite/store.go`)
Table `scheduled_jobs(id, resource_id, [group], message_name, message_data JSON, queued_at, not_before, policy, errcode, retrieved)`.

- **Runners** (`RunnerDefinition{PollInterval, Workers, Messages}`, built from the config in `cmd/config/runners.go`): each has its own
  poller goroutine and workers. The poller claims at most as many jobs as it has idle workers, so `retrieved = true` (shown as
  "running") means a worker has it. It polls on start, on its ticker, and when woken: `Queue` wakes the runner of the message
  through `store.AfterCommit` (`sqlite.AfterCommit`, run by `WriteAndDispatch` after `tx.Commit()`), and a worker wakes its
  runner when it finishes a job. A handler panic is recovered and goes through `Retry` as `job panicked: ...`.
- **Stop:** each runner's `done` is closed; the poller is the only sender on the jobs channel and closes it, workers finish their
  current job. Jobs claimed but not handed over are given back with `Release`.
- **Picking** (`GetNextPendingJobs(ctx, limit, names...)`, an atomic `UPDATE ... RETURNING`): at most **one job per group**, only if
  no job of that group is currently `retrieved`, the job is due (`julianday(not_before) <= julianday('now')`; a plain string
  compare with `DATETIME('now')` skips jobs queued in the current second), and, with the `WaitForOthersResourceID` policy, it is the
  **only** remaining job for its `resource_id`. The group head is chosen across all message types **before** filtering on the
  runner's names, so a runner never jumps ahead of an older job of the same group waiting for another runner. Heads are ordered by
  the raw `not_before` string (nanoseconds; julianday only keeps milliseconds). The default group is the job id (no serialization).
- **Policies (bit flags):** `1 RetryPreserveOrder` (on error, shift the whole group by +14+n seconds), `2 WaitForOthersResourceID`,
  `4 Cancellable` (deletable via `DELETE /api/v1/jobs/:id`), `8 Merge` (if a non-retrieved job with the same resource and name exists, replace its payload instead of inserting).
- **Done:** the row is deleted. **Error:** `Retry` sets `errcode` and `not_before = now+15s`. A job type no runner handles is refused at
  startup.
- SQL trigger `on_deployment_failed_cleanup_jobs`: when a deployment becomes Failed(2), it deletes pending jobs whose
  `resource_id = appId-number`.

Groups used:
- `deployment.deployment.deploy.<appname>-<env>-<appid>` runs deployments of one app+env strictly one at a time, in order.
- `deployment.target.configure.<targetID>` runs one configuration of a target at a time.

### 5.6 Domain model (`internal/deployment/domain`)

**Target** = a host where apps run.
- Fields: id, name, `url Maybe[Url]` (nil means a **manual** proxy), provider config, state, customEntrypoints, cleanupRequested, created.
- State: `Configuring(0) → Ready(2) | Failed(1)`. `version` is a timestamp set on every reconfigure, and `Configured(version, ...)`
  ignores **outdated** results. `lastReadyVersion` remembers whether the target was ever ready (used by the cleanup strategy).
- Uniqueness: the `url` is unique, and `provider_fingerprint` is unique. For docker the fingerprint is the SSH host, and `""` for local,
  so there is **one local target and one target per remote host**.
- Changing the provider kind or host is forbidden (`target_provider_update_not_permitted`).
- `customEntrypoints`: `map[AppID]map[Env]map[EntrypointName]Maybe[Port]`. A `None` port means it isn't allocated yet.
- Deletion: `RequestCleanup` (errors if apps still use the target or it is configuring) → cleanup job → delete job.

**App**
- `name` matches `^[a-z0-9-_]+$` and must **not end with `-staging`**. It is unique per (name, production_target) and (name, staging_target).
- `production` / `staging` `EnvironmentConfig{target, version, vars map[service]map[KEY]VALUE}`.
  `version` is set when the target changes (it is kept when only vars change) and defines the time window
  for "were there successful deployments on that target?".
- `versionControl Maybe{url, token Maybe}` is required for git deployments.
- Deletion: `RequestCleanup` → jobs → `Delete(true)`.

**Deployment** (id = `(appID, number)`, number = `COUNT(*)+1`)
- `ConfigSnapshot` is frozen at creation: appid, appname, environment, target, vars.
- `SourceData`: `raw` (string compose content), `archive` (temp file path), `git` `{branch, hash}`.
- State: `Pending(0) → Running(1) → Failed(2) | Succeeded(3)`, with errcode, `services` (JSON), startedAt, finishedAt.
- Created only via `app.NewDeployment`, `app.Redeploy(src)` or `app.Promote(src)` (staging → production; reuses the source).

**Registry**: name, unique url, optional credentials (username/password). Used by **every** deployment.

**Services / Entrypoints** (`service.go`): built by `ServicesBuilder` during a deployment.
- Entrypoint name: `<project>-<service>-<port>-<router>` with router `http|tcp|udp`.
- The **first http port of the first service in alphabetical order** gets the default subdomain. Other services' first http port gets
  `<service>.<subdomain>`. Additional http ports and all tcp/udp ports are **custom** entrypoints (a dedicated port allocated on the target).

Naming helpers (`config_snapshot.go`):
- project: `<appname>-<env>-<lower(appid)>`
- built image: `<appname>-<lower(appid)>/<service>:<env>`
- subdomain: `appname` (production) / `appname-staging`; non-default services get `service.` prefixed.

**Auth domain**: `User{id, email, bcrypt password hash, apiKey (64 random alnum chars), registeredAt}`. Only one user is
actually used: the first account (there is no user-management UI).

### 5.7 Event → reaction wiring (`internal/deployment/infra/mod.go`)

| Event | Handlers (in order) |
|---|---|
| `DeploymentCreated` | `deploy.OnDeploymentCreated` → queue `deploy` (group per app+env, `RetryPreserveOrder`) |
| `AppEnvChanged` | `redeploy` (re-queue the latest deployment of that env; silently skipped if impossible) · `cleanup_app` (if target changed: queue cleanup of old target, `Cancellable`) · `fail_pending_deployments` (if target changed: fail pending on old target, `app_target_changed`) · `configure_target` (if target changed: old target `UnExposeEntrypoints(app, env)`) |
| `AppCleanupRequested` | `delete_app` (queue, `WaitForOthersResourceID`) · `cleanup_app` (queue prod + staging cleanup, `Cancellable`) · `fail_pending_deployments` (`app_cleanup_requested`) · `configure_target` (un-expose the app from both targets) |
| `TargetCreated` | `configure_target` queue (group per target, `Merge`) |
| `TargetStateChanged` | if it went to Configuring → queue `configure_target` (`Merge`) |
| `DeploymentStateChanged` | if Succeeded → `target.ExposeEntrypoints(app, env, services)` → may raise `TargetEntrypointsChanged` + reconfigure |
| `TargetCleanupRequested` | `fail_pending_deployments` · `cleanup_target` (`Cancellable`) · `delete_target` (`WaitForOthersResourceID`) |

---

## 6. Feature walkthroughs (end to end)

### 6.1 Login / sessions / API key
- `POST /api/v1/sessions {email,password}` → `login` (bcrypt compare; failures become a validation error on both fields).
  Stores `checkship-session=<uid>` in the cookie session named **`checkship`** (HttpOnly, SameSite=Strict, Secure per config).
- `authenticate(withApiAccess)` middleware: cookie first; if allowed, `Authorization: Bearer <apiKey>` is looked up by
  equality in `users.api_key`. The user id is attached to the ctx via `auth.WithUserID`. Every command reads `auth.CurrentUser(ctx)`.
- `PUT /profile/key` → `refresh_api_key`. `PATCH /profile` → `update_user` (checks email uniqueness).
- `DELETE /session` clears the cookie.

### 6.2 Create / update a target
`create_target`: validates the name and optional url, then `provider.Prepare(payload)` (docker: host via
`ssh.ParseHost`, user default `docker`, port default 22, private key parsed by `x/crypto/ssh`). It checks url and
fingerprint availability, runs `NewTarget` (state Configuring), and calls `ExposeServicesAutomatically(url)` if a url was given
(the url is reduced to its root). Writing emits `TargetCreated` → a configure job is queued.

`update_target` covers rename, url `Patch` (null means switch to the manual proxy), and provider changes (the private key is kept if omitted).
URL and provider changes trigger `reconfigure()`, i.e. a new version and a new configure job (merged into a pending one).
`POST /targets/:id/reconfigure` forces it.

### 6.3 Target configuration job (`configure_target` → `docker.Setup`)
1. Skip if the target is gone or the version is outdated.
2. Remote target: upsert `~/.ssh/config` (`Host <host>  # <targetID>`, `StrictHostKeyChecking accept-new`,
   `User`, `Port`, `IdentityFile ~/.ssh/<targetID>`, `IdentitiesOnly yes`).
3. Connect: docker CLI with `ssh://<host>` (remote) or the local socket.
4. **Manual target:** `compose down` the proxy project (with volumes and images), and that's it.
5. **Automatic:** build the proxy compose project `checkship-internal-<lower(targetID)>`:
   - network `default` named `checkship-gateway-<lower(targetID)>`, label `app.checkship.target=<id>`
   - service `proxy`: image `traefik:v2.11`, restart unless-stopped, docker.sock bind-mounted, port 80
   - flags: `--providers.docker`, `--providers.docker.network=<gateway>`,
     constraints `(Label(app.checkship.target, <id>) && (Label(app.checkship.custom_entrypoints,true) || LabelRegex(app.checkship.subdomain,.+))) || Label(app.checkship.exposed,true)`,
     default rule `Host({{ subdomain label }}.<target host>)`, entrypoint `http` on `:80`.
   - **https url:** entrypoint `insecure :80` redirects to `http`, which now listens on `:443` with the ACME resolver
     `checkship-resolver-<id>` using the **HTTP-01** challenge on `insecure` (not TLS-ALPN-01, which can't pass Cloudflare's proxy),
     storing to the `letsencrypt` volume (`<project>_letsencrypt`). `targets.proxy_revision` + `sqlite.ProxyRevision` (now 1):
     at startup `infra.ReconfigureOutdatedProxies` reconfigures once every automatic target below it. Bump it when a proxy change must
     reach existing targets; new targets are created at the current revision.
   - **Custom entrypoints:** known ports are added as `--entrypoints.<name>.address=:<port>/<proto>` + port mappings.
     Unknown ports start a throwaway `<project>-ports-finder` project (`traefik/whoami`) that publishes container ports
     8080+n with **random host ports**, reads the assigned host ports via `ContainerInspect`, then brings it down. These become the
     entrypoints, and the assignments are returned and saved (`TargetEntrypointsChanged`).
   - Command and ports are sorted so the compose hash stays stable, then `compose up --remove-orphans --wait`.
6. The deferred block re-reads the target, calls `Configured(version, assigned, err)` → Ready/Failed, then Write.

### 6.4 Create / update an app
`create_app` validates the name, optional VCS (url + optional token) and both env targets. `CheckAppNamingAvailability` (target
exists and isn't being cleaned up; name not used on that target) → `NewApp`. `update_app` uses `Patch` semantics for VCS
(null removes it; omitted token keeps it; null token makes it public) and `Maybe` per env. An env change emits `AppEnvChanged`
(see §5.7: auto-redeploy, old target cleanup, and so on).

### 6.5 Queue a deployment (`POST /api/v1/apps/:id/deployments`, API key allowed)
Body (one of):
```jsonc
{ "environment": "production", "raw": "services:\n  app: ..." }
{ "environment": "staging", "git": { "branch": "main", "hash": "optional" } }
// multipart/form-data: environment=production, archive=@project.tar.gz
```
`queue_deployment`: validate env → load app → `source.Prepare`:
- raw: requires non-empty content.
- archive: copies the upload to `$TMPDIR/checkship-archive-*` and stores the path.
- git: needs app VCS; `git ls-remote`-like listing to verify the branch and resolve HEAD when no hash is given
  (errors `git_remote_not_reachable` / `git_branch_not_found`).

Then `GetNextDeploymentNumber` → `app.NewDeployment` → Write → `DeploymentCreated` → `deploy` job queued.
The response is `201` + `Location` + the deployment detail.

### 6.6 Deploy job (`internal/deployment/app/deploy/deploy.go`) — "where the magic happens"
1. Load the deployment (if gone: success, no-op). Load the target (it may be missing).
2. `depl.HasStarted()`. If not pending (e.g. cancelled) → no-op success.
3. If the target is **Configuring** → return `target_configuration_in_progress`, **without** writing. The job is retried (and the
   whole group is shifted, keeping order).
4. Write Running. A deferred block re-reads the deployment, calls `HasEnded(services, err)` and writes it. The job itself
   **succeeds** even if the deployment failed (the error lives in the deployment's errcode).
5. `artifactManager.PrepareBuild`: opens the log file `logs/<ts>-<app>-<n>.deployment.log` (lines `[STEP]/[INFO]/[WARN]/[ERROR]`)
   and empties and recreates the build dir `apps/<app>/<template>`.
6. Missing target → fail. Target Failed or being cleaned up → fail.
7. `source.Fetch`: raw writes `compose.yml`. Archive gunzip+untars into the build dir. Git does a single-branch clone, then
   resolves the revision and checks it out.
8. Load **all registries**.
9. `provider.Deploy` (docker):
   - connect (with registry auth injected into the CLI config file in memory)
   - **find the compose file** in this order: `compose.checkship.<env>.yml|yaml`, `docker-compose.checkship.<env>.*`,
     `compose.<env>.*`, `docker-compose.<env>.*`, `compose.checkship.*`, `docker-compose.checkship.*`, `compose.*`, `docker-compose.*`
   - load the project with name = project name, **profile = env**, and an interpolation hook that records each port
     definition's router: no `/` → http, `/tcp` → tcp, `/udp` → udp. **Ports without a host part are ignored.**
   - transform, iterating services alphabetically:
     - set compose labels; warn if there's no restart policy or a bind mount
     - `build:` services get the image forced to the generated name, `pull_policy: build`, and build labels
     - inject the app env vars for that service (they override the compose values)
     - add labels `app.checkship.application`, `app.checkship.target`, `app.checkship.environment`
     - if the target is automatic and has ports: assign entrypoints, set `app.checkship.subdomain` for the main http entrypoint,
       `app.checkship.custom_entrypoints=true` for custom ones, and traefik router/service labels (`HostSNI(*)` for tcp),
       **remove `ports:`**, and join the external `checkship-gateway-<target>` network
     - label networks and volumes
   - `compose up` (build, remove orphans, wait) → prune **dangling** images with the app/target/env labels.
10. Success → `DeploymentStateChanged(Succeeded)` → the target exposes custom entrypoints → maybe a reconfigure (§6.3).

URLs are **computed at read time** (`get_deployment.ResolveServicesUrls`) from the target's *current* url and assigned ports:
- default http: `<scheme>://<subdomain>.<host>`
- custom: `<router>://<subdomain?>.<host>:<publishedPort>`

Example (`docs/reference/providers/docker.md`): app `sandbox`, target `http://docker.localhost` gives
`app` → `http://sandbox.docker.localhost` and `sidecar` → `http://sidecar.sandbox.docker.localhost`.
In staging it would be `sandbox-staging...`.

### 6.7 Redeploy / promote / logs
- `POST .../deployments/:n/redeploy` → `app.Redeploy` creates a new number with the same source and env (using the **current** app config).
- `POST .../deployments/:n/promote` → only from non-production, re-runs the same source in production.
- `GET .../deployments/:n/logs` serves the raw log file (copy, CI, and the UI's fallback when the stream keeps failing).
- `GET .../deployments/:n/logs/stream` (`cmd/serve/deployment_logs.go`) follows the log as SSE: `logs` events `{text}` with whole lines
  only (64 KiB max), `id` = byte offset (resume with `Last-Event-ID` or `?offset=`), `reset` when the offset is past the end, `end`
  once the deployment has finished and everything was sent. It wakes on `live.Signals` (`pkg/live/signals.go`), which the artifact
  manager notifies with the log path on every write (`artifact.WithLogWritten`, wired in `infra/mod.go`; `startup` creates the
  `Signals` and exposes `LogSignals()`), with a 1 s fallback tick. Its own `live.Limiter` (5 per user, 40 in all). The page
  (`#lib/deployment-logs.svelte.ts`, pure parts in `#lib/deployment-logs.ts`) collapses image download lines per layer for display.
- The docker `progressFilter` keeps a pull/BuildKit progress line on each new tenth **or** after 1 s for that layer, so the log moves
  every second (BuildKit's own plain printer still delays parallel steps up to 10 s; see docs/reference/deployments.md#live-logs).

### 6.7b Cancel a deployment (seelf#7)
- `POST .../deployments/:n/cancel` → `cancel_deployment` (cookie or API key). Pending: `Deployment.Cancel()` → Failed with errcode
  `deployment_cancelled` (the `on_deployment_failed_cleanup_jobs` trigger drops the queued job), then the registry is signalled in case a
  worker already claimed it. Running: `RunningDeployments.Cancel(id)` (`infra/running`, in memory, single process) cancels the job's
  context with cause `ErrDeploymentCancelled`; the deploy job ends it as `deployment_cancelled` (its deferred state write uses
  `context.WithoutCancel`). Finished, or running with no worker: `deployment_not_cancellable` (400).
- The deploy job calls `running.Track` first thing. The trigger only deletes unclaimed jobs, so a running job keeps its group lock until
  the worker has stopped: the next deployment of that app+env never overlaps a cancelled one.
- A cancelled deployment is stored as Failed but is **not a failure**: `get_dashboard.outcome` counts it as "other" (no error code, no
  duration, no recovery window), the activity feed's Failed filter leaves it out, and the UI uses `isCancelled`/`isFailure`
  (`#lib/status.ts`) for the grey Cancelled badge, the `stopped` timeline step and the deploy strip. DESIGN.md §11.4/§11.5/§11.7.

### 6.8 Exposing checkship itself (`EXPOSED_ON`, `expose_checkship_container`)
`EXPOSED_ON=http://checkship@docker.localhost`: the user part is the **container name**, the rest is the default local target url.
At startup: find the local target (`provider_fingerprint=''`). If it exists, just `NetworkConnect` the container to its gateway network.
If not, create target "local" with that url, run `Setup` **synchronously**, mark it Configured, Write, then connect and **restart**
the container (traefik needs the new IP; "forbidden" means it's already attached, so nothing happens). The checkship container must carry
labels `app.checkship.exposed=true` and `app.checkship.subdomain=<sub>` (see the root `compose.yml`).

### 6.9 App deletion
`DELETE /apps/:id` (cookie only) → `RequestCleanup` → (in the tx) pending deployments failed, entrypoints un-exposed,
2× `cleanup_app` jobs (Cancellable) + `delete_app` (waits for the others on the same resource id).
- `cleanup_app`: the strategy is **Skip** if the target is being deleted or there were no successful deployments in the window
  `[env.version, now]`. It **errors** (and retries) if there are running or pending deployments, or the target is configuring or failed.
  Otherwise it removes all containers, volumes, networks and images labeled with app+target+env.
- `delete_app`: `artifactManager.Cleanup` (rm `apps/<id>`, logs `*<id>*.deployment.log`) → `AppDeleted` → row deleted
  (deployments cascade).
- If a target is unreachable forever, **cancel** the cleanup job from the Jobs page and the delete then proceeds.

### 6.10 Target deletion
Requires no apps on it and not configuring. `cleanup_target` strategy: running deployments → error; Ready → remove everything
labeled `app.checkship.target=<id>`; never-ready → Skip. `delete_target` → `RemoveConfiguration` (ssh config entry + key file removed) → row deleted.

### 6.11 Registries
CRUD (`/api/v1/registries`). Passwords and tokens are returned masked via `storage.SecretString` (`****`). The UI sends the password
only if it changed. All registries are injected as docker auth for every deployment.

### 6.13 Runtime logs (`GET /api/v1/apps/:id/logs`, API key allowed)
1. `cmd/serve/logs.go` takes a slot in `live.Limiter` (3 per user, 20 in all), then sends `get_app_logs.Query`. The handler resolves the
   app, the environment's target, and returns a `domain.LogsStream` bound to `Provider.Logs`.
2. The docker provider (`docker/logs.go`) lists the containers by the app, target and environment labels (`ErrNoContainers` when none),
   watches `Events` for them, and reads `ContainerLogs(follow, timestamps, tail|since)` per container. `docker/frames.go` turns the stream
   into lines: multiplexed frames (stdout/stderr) or raw TTY output, pieces of lines over 16 KiB joined with their inner timestamps dropped
   (docker/cli#4941), lines cut at 64 KiB, `\r` handled. A `start` event follows the new container (redeploys, restarts).
3. Lines of the first 300 ms are sorted by time, then `consumer.Ready()` is called; the handler only switches to SSE then, so
   `no_containers` (404) and `logs_unavailable` (409) are plain JSON. After that: `logs` batches (100 ms or 500 lines, `id` = last line
   time, used as `since` on reconnect), `state` events, `: ping` every 15 s, `error` if the stream fails.
4. The page (`routes/(main)/apps/[id]/logs/+page.svelte`) reads the stream with `fetch` through `LogsConnection`
   (`#lib/runtime-logs.svelte.ts`) to see HTTP statuses; parsing, the 5,000-line buffer, dedupe after reconnect, search (`-word`
   hides) and the copy formats are in `#lib/runtime-logs.ts` (unit tested). Copy goes through `#lib/clipboard.ts` (`copyText`, with
   a textarea fallback over plain HTTP).
5. Streams end with the request or on shutdown: `server.shutdown` is cancelled from `srv.RegisterOnShutdown`, because `Shutdown` waits for
   open requests. `GET /apps/:id/services` (`get_app_services`) returns the container states for the Logs tab's strip.

### 6.14 Custom domains (seelf#5)
- Model: `domain.Hostname` (`hostname.go`: lowercase, punycode via `x/net/idna`, RFC 1123 labels, ≥2 labels, no scheme, port, path,
  wildcard or IP; only `[a-z0-9.-]`, so it is safe inside Traefik's backtick rules). `ServicesDomains map[service][]Hostname` lives in
  `EnvironmentConfig.domains` (compared by `Equals`, so a change raises `AppEnvChanged` and the usual redeploy) and is frozen in
  `ConfigSnapshot.domains` (`deployments.config_domains`).
- Rules (`create_app.BuildDomains`, used by create and update): valid, ≤20/service, ≤50/environment, unique in the app, not under the
  target's host (`domain_reserved_by_target`), not `EXPOSED_ON`'s host (`domain_reserved`), automatic proxy only
  (`custom_domains_need_automatic_proxy`), not used by another app (`domain_already_used` + `DomainOwner{app}` detail, via
  `AppsReader.GetDomainsOwners`). `app_domains(hostname PK, app_id CASCADE, environment, service)` is kept in sync by the apps store
  and backs uniqueness. Update keeps an environment's domains when `domains` is omitted and re-checks both environments.
- Deploy (`docker/domains.go`): one router per domain on the service's main http entrypoint
  (`<entrypoint>-d<sha1[:4] hex>`, rule `Host(...)`, entrypoint `http`, service = the entrypoint's), so each domain gets its own
  certificate. Ignored domains (manual proxy, unknown service, no http port) are `[WARN]`ed in the deployment log. `Service.domains`
  is stored in `state_services`; `get_deployment` returns `domains` and `urls` (custom first, then default).
- Check: `GET /apps/:id/domains/check` → `check_app_domains` → `infra/domaincheck`: A/AAAA/CNAME (system resolver, 3s), CAA via
  `x/net/dns/dnsmessage` to `/etc/resolv.conf` nameservers (closest set up the tree), Cloudflare ranges (`cloudflare.go`, dated), and a
  TLS handshake **only to an address the target's default host resolves to** (pinned IP, SNI = domain, no HTTP). Cached 30s; its own
  limiter (2 per user). Front: `#lib/hostname.ts` (validation mirror, chips, DNS records), `routes/(main)/apps/domains-drawer.svelte`.
- Connect Cloudflare (`domain/cloudflare.go`, `app/cloudflare_setup`, `infra/cloudflare` API v4 client, `cloudflare_connections` table):
  token from a pre-filled link (`#lib/cloudflare.ts`: zone read, dns edit, argotunnel edit), `PUT/GET/DELETE /targets/:id/cloudflare`,
  `POST …/setup` (session only). Sync: tunnel `checkship-<target>` + ingress (final `http_status:404`), proxied CNAMEs with comment
  `managed by checkship (target <id>)` (the only records it touches; others are `conflicts` unless in `replace`), connector container
  `checkship-tunnel-<target>` (`docker/tunnel.go`, pinned `cloudflare/cloudflared` image) on the gateway network. App created/env
  changed/cleanup queue `SyncJob` (group `deployment.cloudflare.<target>`, Merge); target cleanup queues `DisconnectCommand`.
  `domaincheck` sets `cloudflare_dns` from the closest NS set (`*.ns.cloudflare.com`), which shows the drawer section.
  Tokens: `cfat_` (account) verified on `/accounts/{id}/tokens/verify` (account from `/accounts`, else zones), others on
  `/user/tokens/verify` first. Connect can only probe tunnel Read; a refused create sets `tunnel_allowed=false`, mode dns, and a
  later successful sync sets it back. Conflicts carry `want` (type/name `@`/content/proxied) and `reason` (`record`, `dns_edit` when
  the token can't write DNS: the record is added by hand). A matching hand-made CNAME is taken over (comment added), or accepted
  as is when proxied and DNS is read-only. Permission errors keep Cloudflare's message (`detail.cloudflare`).

### 6.12 Jobs page
`GET /api/v1/jobs?page=n` (10 per page). Shows status (retrieved = running, errcode = failed, otherwise pending), payload, policy and
error. A cancel button appears only for `Cancellable` jobs (`DELETE /jobs/:id`; the store refuses non-cancellable ones with a 404).

---

## 7. HTTP API (`cmd/serve/server.go`) — base `/api/v1`

| Method & path | Auth | Handler → message |
|---|---|---|
| POST /sessions | public | login (+ get_profile) |
| GET /healthcheck | public | `{version}` |
| DELETE /session | cookie | clear session |
| GET /jobs ?page | cookie | ScheduledJobsStore.GetAllJobs |
| DELETE /jobs/:id | cookie | cancel (Cancellable only) |
| GET/PATCH /profile, PUT /profile/key | cookie | get_profile / update_user / refresh_api_key |
| POST /targets, GET /targets ?active_only, GET/PATCH/DELETE /targets/:id, POST /targets/:id/reconfigure | cookie | create/get/update/request_target_cleanup/reconfigure |
| POST/GET /registries, GET/PATCH/DELETE /registries/:id | cookie | registry CRUD |
| GET /apps, POST /apps, PATCH /apps/:id, DELETE /apps/:id | cookie | get_apps/create/update/request_app_cleanup |
| GET /apps/:id | cookie **or API key** | get_app_detail (env `domains` included) |
| GET /apps/:id/domains/check ?environment | cookie or API key | check_app_domains (DNS, CAA, certificate; 429 when busy) |
| POST /apps/:id/deployments | cookie or API key | queue_deployment (json or multipart) |
| GET /apps/:id/deployments ?page&environment | cookie or API key | get_app_deployments (5/page) |
| GET /apps/:id/deployments/:n | cookie or API key | get_deployment |
| POST /apps/:id/deployments/:n/redeploy, /promote | cookie or API key | redeploy / promote |
| POST /apps/:id/deployments/:n/cancel | cookie or API key | cancel_deployment (400 `deployment_not_cancellable`) |
| GET /apps/:id/deployments/:n/logs | cookie or API key | raw log file |
| GET /apps/:id/deployments/:n/logs/stream ?offset | cookie or API key | deployment log as SSE (§6.7); 429 `too_many_streams` |
| GET /apps/:id/logs ?environment&tail&since | cookie or API key | get_app_logs → SSE stream (§6.13); 404 `no_containers`, 409 `logs_unavailable`, 429 `too_many_streams` |
| GET /apps/:id/services ?environment | cookie or API key | get_app_services → container states |

Errors: `apperr.Error{code, detail}`. App errors → **400** (404 for `not_found`). Bind errors → 422. Anything else → 500
`{code:"unexpected_error"}` (logged). Validation errors are `validation_failed` with `detail` = `{ "field.path": {code} }`,
which the frontend maps to form fields. Creates return 201 + `Location`. Responses carry `Cache-Control: public, max-age=0, must-revalidate`.
Static `.js/.css` get immutable caching for a year.

---

## 8. Database schema (current)

```
users(id PK, email UQ, password_hash, api_key UQ, registered_at)
targets(id PK, name, url NULL UQ, provider_kind, provider_fingerprint UQ, provider JSON,
        state_status, state_version, state_errcode, state_last_ready_version, entrypoints JSON default '{}',
        cleanup_requested_at/by, created_at/by→users)
apps(id PK, name, version_control_url, version_control_token,
     production_target→targets RESTRICT, production_version, production_vars JSON, production_domains JSON,
     staging_target→targets RESTRICT, staging_version, staging_vars JSON, staging_domains JSON,
     created_at/by, cleanup_requested_at/by,
     UQ(name, production_target), UQ(name, staging_target))
deployments(PK(app_id→apps CASCADE, deployment_number), config_appid, config_appname, config_environment,
            config_target (no FK on purpose), config_vars JSON, config_domains JSON, state_status, state_errcode, state_services JSON,
            state_started_at, state_finished_at, source_discriminator, source, requested_at, requested_by→users)
            idx(config_target), idx(state_status); TRIGGER on_deployment_failed_cleanup_jobs
registries(id PK, name, url UQ, credentials_username, credentials_password, created_at/by)
app_domains(hostname PK, app_id→apps CASCADE, environment, service)  idx(app_id)   -- targets also have proxy_revision
scheduled_jobs(see §5.5)  idx(resource_id), idx([group])
auth_schema_migrations, deployment_schema_migrations, scheduler_schema_migrations
```
History notes: v1→v2 migration `1706004450_add_target` created a default target with the **fixed id
`2bRUdQnyRELMqyh9gFLQV1s0cqv`** and moved apps onto it (documented in `docs/guide/migration.md`).

---

## 9. Frontend (`cmd/serve/front`)

- SPA: `+layout.ts` sets `ssr=false`, `prerender=false`, `trailingSlash='always'`. adapter-static `fallback: 'fallback.html'`.
  The Go side serves files from the embed FS and falls back to `fallback.html`.
- Imports: subpath imports `#lib/*` → `src/lib/*` (package.json `imports`); there are no `config.alias` entries.
- UI: Svelte 5 + SvelteKit 3, shadcn-svelte on Bits UI, Tailwind 4. `DESIGN.md` is the spec; the theme is `src/routes/layout.css`.
- Lists: there are no tables. Every list is `#lib/components/timeline/` rows (`timeline-row.svelte`) in a
  `timeline-group.svelte` (grouped by day or state, `lib/timeline.ts`) or a plain `timeline-list.svelte`; DESIGN.md §5 Lists.
- Cardless (DESIGN.md §1): the workspace is the one container. No grey boxes around sections, notes or settings; use
  `section-heading.svelte`, `notice.svelte` (tinted only for warning or danger) and `setting-row.svelte`. `bg-sunken` is
  only for inputs, choice cards, the logs console and error text (row icons are bare since 2026-11-20).
- Empty states: `empty-state.svelte` with an `illustration` from `illustration.svelte` and a `size` (page, section,
  compact); never a grey line or a boxed blank slate. DESIGN.md §11.5.
- Charts: drawn in SVG by `#lib/components/charts/` (column, line, share bar, calendar, metric) with the maths in
  `#lib/charts.ts`; no chart library. Backups figures are counted in `#lib/backup-stats.ts`. DESIGN.md §11.10.
- **Data layer** `src/lib/fetcher`: `CacheFetchService`. `get` and `query` share a cache keyed by url+params, with 2s dedupe.
  `query` returns Svelte stores `{data, error, loading}` with optional polling (`refreshInterval`). Mutations
  (`post/patch/put/delete`) invalidate their url plus extra keys and call SvelteKit `invalidate`.
- **Resources** `src/lib/resources/*.ts` are thin typed services per API resource (apps, deployments, targets, registries,
  jobs, sessions, users, healthcheck). Polling: 5s in general (`POLLING_INTERVAL_MS`), 2.5s for running deployments.
- **Auth**: `(main)/+layout.ts` calls `GET /profile` (`auth.checkSession`). On failure it logs out and redirects to
  `/signin?redirectTo=...`. Login resets the fetch cache and redirects.
- **Forms**: `lib/form.ts submitter()` provides the loading state and the error mapping (`BadRequestError.fields` from
  `validation_failed`); confirmations use `#lib/components/confirm-dialog.svelte`.
- **i18n**: `lib/localization` (en, fr). Typed keys; error codes and job message names are translation keys. The locale is
  stored in `localStorage['locale']` and changing it reloads the page.
- **cURL helper**: `lib/curl.ts` builds the CI command shown on the "New deployment" page.
- **Routes**:
  - `(auth)/signin`
  - `(main)/` (apps list), `apps/new`, `apps/[id]` (env cards + paginated history), `apps/[id]/edit` (settings + delete)
  - `apps/[id]/deployments/new` (env, kind raw|archive|git, curl panel), `apps/[id]/deployments/[number]` (detail, logs console, redeploy, promote)
  - `targets`, `targets/new`, `targets/[id]/edit` (status panel, reconfigure, delete)
  - `registries`, `registries/new`, `registries/[id]/edit`
  - `jobs` (list grouped by state + cancel), `profile` (email, password, locale, API key)
- **Design tokens** in `src/routes/layout.css` (DESIGN.md §3 palette, light and dark): CSS variables mapped to Tailwind with
  `@theme inline`. Tailwind's preflight is the only reset; the old `src/assets/styles.css` and `src/components/` are gone.
- Tests: `vitest` on `lib` (fetcher cache, curl, select, localization).

---

## 10. Conventions for changes

- **New use case:** create `internal/<ctx>/app/<snake_name>/<snake_name>.go` with a `Command`/`Query` struct embedding
  `bus.Command[T]`/`bus.Query[T]`, a `Name_()` of the form `"<ctx>.command.<name>"`, and `Handler(deps...) bus.RequestHandler[T, Command]`.
  Register it in `internal/<ctx>/infra/mod.go`. Expose it via a gin handler in `cmd/serve/*.go` + a route in `server.go`.
  Add a test using `fixture.PrepareDatabase(t, fixture.WithUsers(...), ...)` and `pkg/assert`.
- **Background job:** the command also implements `ResourceID()`. Queue it from an `On<Event>Handler(scheduler)` with
  `bus.WithGroup` / `bus.WithPolicy`. The catch-all runner (last runner without `jobs`) picks it up; if the runners config
  lists every job explicitly, startup fails until the new name is added. List it in `docs/guide/configuration.md#runners` and add a
  translation key named after the message for the Jobs page.
  ⚠ `message_name` values are **persisted** in `scheduled_jobs`, so renaming them breaks queued jobs.
- **Entity change:** add an event struct + `Name_()`, mutate only through `apply`, handle the event in the store's `Write` switch.
- **Optional values:** use `monad.Maybe[T]`. For "absent vs null vs value" use `monad.Patch[T]`. Avoid pointers.
- **Validation:** `validate.Struct(validate.Of{"field": validate.Value(raw, &target, factory)})`. Field keys become frontend error keys.
- Domain errors are `apperr.New("snake_code")` (shown to users, translated in `en.ts`/`fr.ts`). Infra errors use plain `errors.New`.
- **New provider or source:** implement `provider.Provider` (`CanPrepare/CanHandle` + domain interface) or `source.Source`, register the
  data type in `init()` with the DiscriminatedMappers, and add it to the facade in `infra/mod.go`. Add a JSON body field in `cmd/serve`.
- **Migration:** `internal/<ctx>/infra/sqlite/migrations/$(make ts)_name.up.sql`. SQLite can't alter constraints, so use the
  temp-table copy pattern seen in existing migrations.
- Frontend style: tabs, single quotes, width 100 (`.prettierrc`). `npm run check` must pass (CI).

---

## 11. Rename from seelf to checkship (done)

The project was forked from **seelf** (v2.4.2, commit `d4703a3`) and renamed to **checkship**. Every identifier was
renamed because checkship is a new product with no existing installations to keep compatible.

| Before | After |
|---|---|
| Go module `github.com/YuukanOO/seelf` | `github.com/checkship-deploy/checkship` |
| Binary / CLI `seelf` | `checkship` |
| Docker image `yuukanoo/seelf` (Docker Hub) | `ghcr.io/checkship-deploy/checkship` (GitHub Container Registry) |
| Docs `yuukanoo.github.io/seelf`, VitePress `base: "/seelf/"` | `checkship-deploy.github.io/checkship`, `base: "/checkship/"` |
| Env `SEELF_ADMIN_EMAIL/PASSWORD`, `SEELF_VERSION` | `CHECKSHIP_ADMIN_EMAIL/PASSWORD`, `CHECKSHIP_VERSION` (`ADMIN_*` still works) |
| Docker labels `app.seelf.*` | `app.checkship.*` |
| Target network / proxy project / ACME resolver `seelf-gateway-<id>` / `seelf-internal-<id>` / `seelf-resolver-<id>` | `checkship-gateway-<id>` / `checkship-internal-<id>` / `checkship-resolver-<id>` |
| DB file `seelf.db`, data dir `~/.config/seelf`, container path `/seelf/data` | `checkship.db`, `~/.config/checkship`, `/checkship/data` |
| Session cookie `seelf`, key `seelf-session` | `checkship`, `checkship-session` |
| Compose file names `compose.seelf[.<env>].yml` … | `compose.checkship[.<env>].yml` … |
| Package `internal/deployment/app/expose_seelf_container` | `expose_checkship_container` |
| Git basic-auth user, temp archive prefix | `checkship`, `checkship-archive-` |

Kept on purpose:
- `go.mod` `replace` lines to `github.com/YuukanOO/compose/v2` and `github.com/YuukanOO/ssh_config` (third-party forks the build depends on).
- Applied SQL migrations (e.g. the `seelf_incompatible_upgrade` error code in `1702544287_delete_worker_stuff.up.sql`). Never edit migrations.
- `LICENSE` (GPL-3.0) and credit to the original author (README, `docs/contributing/credits.md`, dashboard footer "Credits").
- `api.http` sample git URL (a public sample repository by the original author, used only in that scratch file).
- `docs/guide/migration.md` describes seelf v1 → v2 and was only renamed mechanically. It doesn't apply to checkship installs.
- Binary archives in `examples/*/*.tar.gz` still contain the old name inside their sources; rebuild them with each example's `Makefile` if needed.

Verify nothing was missed:
```sh
git ls-files | grep -vE 'package-lock|go\.sum|AGENT\.md' | xargs grep -nIi 'seelf\|yuukanoo\|leicher'
# expected: go.mod replace lines, the old migration, api.http, credits page, README credit
```

## 12. Repository, CI and license

- Remote: `origin` = `https://github.com/checkship-deploy/checkship.git` (private). The upstream link to `YuukanOO/seelf` was removed; history is kept.
- CI (`.github/workflows/ci.yml`) on every push: backend (vet + tests), frontend (check + tests), docs build, docs deploy to GitHub Pages (main only),
  release (main/next): semantic-release with the built-in `GITHUB_TOKEN`, then a multi-arch image pushed to `ghcr.io/checkship-deploy/checkship`.
  No extra secrets are needed. The original author's CodeClimate coverage upload and Docker Hub login were removed. `.github/FUNDING.yml` was removed.
- GitHub Pages on a **private** repository requires a paid GitHub plan for the organization. Without it the `deploy-docs` job fails, which doesn't block anything else.
- **License:** GPL-3.0, unchanged. checkship is a derivative work of seelf, so it can't be relicensed (e.g. to MIT or proprietary)
  without permission from the copyright holder (Julien Leicher). checkship can be modified, renamed, sold and hosted, but anyone who receives the
  binary or image must be able to get the complete source under GPL-3.0, and the license and credits must stay.

## 13. Brand: the checkship logo

Sources of truth (repository root):
- `assets/brand/logo-mark.svg`: the mark, 24-unit grid, stroke 2. Blue switches with `prefers-color-scheme`.
- `assets/brand/logo.svg`: the lockup, with the wordmark outlined (Inter SemiBold), viewBox `0 0 119.7 24`.
- `assets/brand/icon.png` (512px) and `icon.ico` (16–256): generated by `scripts/build-icons.mjs` (`npm run build:icons`, uses `@resvg/resvg-js`).

Geometry: an open ring `M15.37 3.66A9 9 0 1 0 20.56 9.22` (centre 12,12, r 9, open from 18° to 68°) and a check
`M7.25 11.75 10.75 15.25 20 4` whose long stroke leaves the ring through the opening. Round caps and joins.

Rules:
1. Stroke only. No background tile, no fill, no gradient, no shadow.
2. Stroke colour is the brand blue: `#2f5bea` in light, `#7f95ff` in dark. It does not follow the accent tone. The dashboard keeps its emerald accent (`--co-primary-*`), and the logo uses `--co-brand`.
3. Wordmark "checkship" in Inter 600, lower case, in ink, tracking −0.02em, font size 0.75× the mark.
4. The mark sits left of the wordmark with a gap of 0.4× the mark size.
5. Sizes (mark height): 22px title bar (dashboard top bar), 28px sidebar header (docs nav, `--vp-nav-logo-height`), 56px welcome screen (sign-in page), 72px Settings › About (no such screen yet).
6. App icon: the same mark at stroke 2.25 on a 512px transparent canvas, in `#2f5bea`.

Where it's used:
- Dashboard: `cmd/serve/front/src/assets/logo.svelte` (props `size`, `markOnly`), `static/favicon.svg` (= `logo-mark.svg`).
  Inter 600 is bundled via `@fontsource/inter` (`latin-600.css` imported in `src/routes/+layout.svelte`).
- Docs: `docs/public/logo-light.svg`, `logo-dark.svg` (fixed colours per theme), `favicon.svg`.
- README: `<picture>` with the docs light/dark logos.
- Docs theme (`docs/.vitepress/theme/custom.css`): the brand colours are the dashboard accent (`#2f5bea`, dark `#7f95ff`).
- `docs/public/screenshots/*-{light,dark}.webp` are the quickstart screenshots (1440×900, WebP q82), shown per theme with
  `class="screenshot light-only"` / `dark-only`. Retake them when the dashboard changes visually: a throwaway instance with
  a `local` target and a `welcome` app (compose `traefik/whoami`), Firefox at 1440×900 ×2, both themes. The New target and
  first-run shots come from a fresh instance with no target, so **This machine** is available.


## 14. Known issues, risks and tech debt (observed, not fixed)

1. **Archive extraction does not sanitize paths** (`source/archive/source.go`): `filepath.Join(dir, header.Name)` lets
   `../` entries write outside the build dir (zip-slip). Uploaders are authenticated, but this is still worth fixing.
2. Uploaded archives in `$TMPDIR/checkship-archive-*` are never deleted.
3. Secrets are stored in **plaintext** in SQLite: VCS tokens, registry passwords and SSH private keys (in the `provider` JSON
   and on disk at `~/.ssh/<id>`). The API key is stored plaintext and compared by equality.
4. `GetNextDeploymentNumber` = `COUNT(*)+1`, which can race (FIXME in code). Mitigated by the immediate tx lock.
5. Entity events are never cleared after `Write` (TODO #71).
6. `monad.Maybe.Scan` can panic on type mismatch (FIXME).
7. It's effectively single-user: no user management, and every authenticated user can do everything.
8. Only the deployment routes accept API keys (issue #45). App deletion and targets need the cookie.
9. A shared proxy for default and custom entrypoints causes a brief downtime when entrypoints change (issue #62).
10. Version skew: CI uses Go 1.21 but go.mod says 1.23. The Dockerfile uses Node 18 while CI uses Node 20.
11. Failed jobs retry forever every 15s, with no max attempts.
12. `get_deployment` keeps backward-compat for pre-2.2 service formats (`Url` / `Subdomain` fields).

---

## 15. Glossary

- **Target**: a Docker host (local socket or SSH) + optional root URL. URL means the automatic Traefik proxy; no URL means a manual proxy.
- **App**: a named stack with production and staging environment configs (target + per-service env vars) and optional VCS.
- **Deployment**: a numbered run of an app in an environment from a source (raw, archive or git), with frozen config, state and services.
- **Entrypoint**: an exposed port of a service. Default http goes through the proxy's 80/443. Custom (extra http, tcp, udp) gets a dedicated allocated port.
- **Registry**: a private image registry with optional credentials, applied to all deployments.
- **Job**: a persisted background command processed by the scheduler workers.
- **Cleanup**: removal of checkship-labeled Docker resources from a target, for one app+env or for a whole target.
