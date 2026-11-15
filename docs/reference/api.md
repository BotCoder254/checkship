# API

This part is currently [being worked on](https://github.com/checkship-deploy/checkship/issues/45). For now, only routes related to the deployment can be accessed with the API Key retrieved from the **profile page**. For more information, you can check the [`api.http` file](https://github.com/checkship-deploy/checkship/blob/main/api.http) in the source code.

Every other routes use a cookie authentication.

## Allowed API access routes

The following routes are allowed with an header `Authorization: Bearer <user API Key>`.

```http
# Retrieve an app details
GET /apps/:id
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
# Retrieve deployment logs
GET /apps/:id/deployments/:number/logs
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
