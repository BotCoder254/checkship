# API

This part is currently [being worked on](https://github.com/checkship-deploy/checkship/issues/45). For now, only routes related to the deployment can be accessed with the API Key retrieved from the **profile page**. For more information, you can check the [`api.http` file](https://github.com/checkship-deploy/checkship/blob/main/api.http) in the source code.

Every other routes use a cookie authentication.

## Allowed API access routes

The following routes are allowed with an header `Authorization: Bearer <user API Key>`.

```http
# Retrieve an app details
GET /apps/:id
# DNS, CAA and certificate status of an environment's custom domains
GET /apps/:id/domains/check?environment=production
# Creates a new deployment
POST /apps/:id/deployments
# Get all deployments of an app
GET /apps/:id/deployments
# Get a specific app deployment
GET /apps/:id/deployments/:number
# Redeploy a deployment
POST /apps/:id/deployments/:number/redeploy
# Promote a staging deployment to the production environment
POST /apps/:id/deployments/:number/promote
# Cancel a pending or running deployment
POST /apps/:id/deployments/:number/cancel
# Retrieve deployment logs
GET /apps/:id/deployments/:number/logs
# Follow a deployment log while it is written (server-sent events)
GET /apps/:id/deployments/:number/logs/stream?offset=0
# Follow the logs of an application's containers (server-sent events)
GET /apps/:id/logs?environment=production&tail=200
# State of an application's containers
GET /apps/:id/services?environment=production
# Named volumes of an environment, and the mounts a backup leaves out
GET /apps/:id/volumes?environment=production
# Back up, list, upload, download, restore and delete backups
GET /apps/:id/backups
POST /apps/:id/backups
POST /apps/:id/backups/upload
GET /backups
GET /backups/:id
GET /backups/:id/download
POST /backups/:id/restore
POST /backups/:id/store
POST /backups/:id/retry
DELETE /backups/:id
```

The [Backups](/reference/backups#api) page describes their bodies.

## Custom domains {#custom-domains}

`POST /apps` and `PATCH /apps/:id` accept each environment's [custom domains](/reference/applications#custom-domains) next to its target and variables, per service:

```json
{
  "production": {
    "target": "<target id>",
    "domains": { "web": ["shop.acme.com", "www.shop.acme.com"], "api": ["api.acme.com"] }
  }
}
```

On `PATCH`, an environment sent without `domains` keeps its domains, and `{}` removes them. Errors are `validation_failed` with a field per domain (`production.domains.web.0`): `invalid_hostname`, `domain_duplicated`, `domain_reserved_by_target`, `domain_reserved`, `domain_already_used` (its detail names the application, `{"app": "blog"}`), `too_many_domains` and `custom_domains_need_automatic_proxy`. `GET /apps/:id` returns them under `production.domains` and `staging.domains`.

`GET /apps/:id/domains/check?environment=production` checks the saved domains (cookie or API key; a few checks at a time per user, `429` otherwise):

```json
{
  "environment": "production",
  "https": true,
  "expected": { "host": "shop.apps.acme.com", "addresses": ["203.0.113.10"] },
  "domains": [
    {
      "service": "web",
      "hostname": "shop.acme.com",
      "unicode": "shop.acme.com",
      "dns": "points_here",
      "addresses": ["203.0.113.10"],
      "caa": "caa_ok",
      "certificate": "certificate_ok",
      "issuer": "Let's Encrypt",
      "expires_at": "2027-02-18T10:00:00Z"
    }
  ]
}
```

`dns` is `points_here`, `behind_cloudflare`, `points_elsewhere`, `no_record` or `unknown`; `cloudflare_dns` is `true` when the domain's nameservers are Cloudflare's; `caa` is `caa_ok`, `caa_blocks` (with `caa_issuers`) or `caa_unknown`; `certificate` (https targets, domains pointing here) is `certificate_ok`, `default_certificate` or `unreachable`. Deployment services list their `domains` and their `urls`, custom domains first.

## Connect Cloudflare {#cloudflare}

Per target, with a session cookie only (the token gives access to a Cloudflare account, so an API key can't set it). See [Guide › Cloudflare](/guide/cloudflare#connect).

```sh
# Status: account, mode, tunnel and the records checkship manages ({"connected": false} when not connected)
GET /targets/:id/cloudflare
# Connect with an API token (Zone Read, DNS Edit, optionally Cloudflare Tunnel Edit)
PUT /targets/:id/cloudflare          {"token": "…"}
# Set everything up for the target's domains; mode "tunnel" or "dns", replace: hostnames whose own record may be replaced
POST /targets/:id/cloudflare/setup   {"mode": "tunnel", "replace": []}
# Remove the tunnel, its connector and checkship's records, then forget the token
DELETE /targets/:id/cloudflare
```

The token is never returned. `setup` answers `{"tasks": [{"key": "tunnel", "detail": "…"}], "conflicts": [{"hostname", "type", "content", "zone", "want": {"type", "name", "content", "proxied"}}], "outside": ["…"], "status": {…}}`: `conflicts` are records checkship didn't create (left alone unless listed in `replace`; `want` is the record to put instead, and one made by hand that matches it is taken over), `outside` domains in no zone of the token's account. Errors: `cloudflare_token_invalid`, `cloudflare_token_refused`, `cloudflare_permission_missing` (detail `{"permission": "DNS · Edit"}`), `cloudflare_unavailable`, `cloudflare_not_connected`, `cloudflare_tunnel_not_allowed`, `cloudflare_invalid_mode`, `cloudflare_automatic_proxy_only`; on `PUT`, token errors come as `validation_failed` on the `token` field. Afterwards, adding or removing a domain on the target updates Cloudflare by itself (a `deployment.command.sync_cloudflare` job).

## Cancel a deployment {#cancel-a-deployment}

`POST /apps/:id/deployments/:number/cancel` cancels a pending or running deployment (see [Cancel a deployment](/reference/deployments#cancel)) and answers `200` with the deployment as it is now, in the same shape as `GET /apps/:id/deployments/:number`.

```sh
curl -X POST -H "Authorization: Bearer <user API Key>" \
  "https://checkship.example.com/api/v1/apps/<app id>/deployments/<number>/cancel"
```

- A **pending** deployment comes back already cancelled: `state.status` is `2` (failed) and `state.error_code` is `deployment_cancelled`.
- A **running** one comes back still running (`state.status` `1`): it stops within seconds and then reads the same way. Poll the deployment, or follow its log stream until `end`, to know when.
- `400` with `deployment_not_cancellable` when the deployment has already finished (succeeded, failed or cancelled). `404` with `not_found` when it doesn't exist.

## Deployment logs {#deployment-logs}

`GET /apps/:id/deployments/:number/logs` returns the whole log as plain text. `GET /apps/:id/deployments/:number/logs/stream` follows it as [server-sent events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events) while the deployment runs, and closes once it has finished and every line was sent.

```sh
curl -N -H "Authorization: Bearer <user API Key>" \
  "https://checkship.example.com/api/v1/apps/<app id>/deployments/<number>/logs/stream"
```

| Event   | Data                                                                                                                                         |
| ------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| `logs`  | `{"text": "..."}`: the new whole lines, at most 64 KiB per event. The event `id` is the byte offset reached in the log                       |
| `reset` | `{}`: the offset asked for is past the end of the log; what follows starts from the beginning, so drop what you have                          |
| `end`   | `{}`: the deployment has finished and everything was sent. The stream closes                                                                  |
| `error` | `{"code": "logs_failed"}` when the log can't be read. Nothing follows                                                                         |

A comment line (`: ping`) is sent every 15 seconds. To resume after a disconnection, send the last event `id` in a `Last-Event-ID` header, or in an `offset` parameter. Errors before the stream starts are plain JSON: `404` with `not_found`, and `429` with `too_many_streams` past the limits (5 streams per user, 40 in all).

## Runtime logs {#runtime-logs}

`GET /apps/:id/logs` streams the logs of an application's containers as [server-sent events](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events). It reads back `tail` lines of each container (default 200, at most 5000), then follows new ones until the connection closes.

```sh
curl -N -H "Authorization: Bearer <user API Key>" \
  "https://checkship.example.com/api/v1/apps/<app id>/logs?environment=staging&tail=50"
```

| Event   | Data                                                                                                                                                                                     |
| ------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `logs`  | An array of lines: `time`, `service`, `container`, `stream` (`stdout` or `stderr`), `text`, and `truncated` when a line over 64 KiB was cut. The event `id` is the time of its last line |
| `state` | A container change: `time`, `service`, `container`, `action` (`start`, `die`, `oom`, `health`, `destroy`), `exit_code`, `health`                                                         |
| `error` | `{"code": "..."}` when the stream stops on an error. Nothing follows                                                                                                                     |

A comment line (`: ping`) is sent every 15 seconds. To resume after a disconnection, send the last event `id` in a `Last-Event-ID` header, or in a `since` parameter: lines are read from that time on, so the one with that exact time may come again.

Before the stream starts, errors are plain JSON: `404` with `no_containers` when the environment has never been deployed, `409` with `logs_unavailable` when Docker can't read the logs (logging driver), and `429` with `too_many_streams` past the limits (3 streams per user, 20 in all).

`GET /apps/:id/services` returns, for each container of the environment: `service`, `container`, `image`, `state` (`running`, `restarting`, `exited`…), `status` (such as `Up 3 hours`), `health`, `restart_count`, `exit_code` and `started_at`.
