# Deployments

Created from an [application](/reference/applications) for a specific [environment](/reference/applications#environments), represents an actual deployment on a [target](/reference/targets).

## Sources {#sources}

Deployments can be created from a number of sources.

### Archive (`tar.gz`)

An archive containing your project files to be deployed.

### Raw file

A raw file. For example, a `compose.yml` content when using the [Docker provider](/reference/providers/docker).

### Git

A valid **branch** and an optional specific **commit** if the application has been configured with a version control system.

## Cancel a deployment {#cancel}

A deployment that is **pending** or **running** can be cancelled with the **Cancel** button on its page, from its row menu on the **Deployments** tab, or from the [API](/reference/api#cancel-a-deployment). checkship asks for confirmation first.

- **Pending:** the deployment is cancelled at once and never runs. Its job leaves the [Jobs](/reference/jobs) page, and the deployments queued after it run as usual.
- **Running:** the deployment stops wherever it is: fetching the source (Git clone, archive extraction), pulling or building images, or starting the containers. It ends within a few seconds, and its log ends with `[WARN] deployment cancelled, stopping`. The next deployment of the same environment only starts once it has really stopped, so two never overlap.

A cancelled deployment shows as **Cancelled**, in grey. It is stored as failed with the error code `deployment_cancelled`, but it is not counted as a failure: the Overview figures, **Needs attention** and the **Failures** drawer leave it out. The **Failed** filter of the Deployments tab still lists it, with its Cancelled badge.

Images already pulled or built stay in Docker's cache, so the next deployment doesn't download them again.

::: warning
If a running deployment is cancelled while `docker compose up` is already replacing containers, the environment may be left **half updated**: some services on the new version, others on the old one, or stopped. Redeploy the deployment the environment was serving before to go back. A cancel that arrives as the deployment finishes may come too late: the deployment then succeeds.
:::

Only a deployment of this checkship instance can be stopped while it runs. After a restart, deployments that were running are failed with `server_reset`, as before.

## Live logs {#live-logs}

The log of a deployment shows **live** on its page while it runs: each line appears as checkship writes it, with a green **Live** marker next to the heading. The page resumes where it was after a network drop (**Reconnecting…**), and falls back to reloading the whole log every few seconds if the stream keeps failing, for example behind a proxy that buffers responses (see [Behind a reverse proxy](/guide/installation#reverse-proxy)).

- **Image downloads** move every second: a layer that is downloading or extracting writes a line when it reaches another tenth of its size, and at least once a second while it moves. On the page, the lines of each layer collapse into one line that updates in place, the way the Docker CLI shows them in a terminal. **Copy** copies the full log, every line included.
- **Image builds** (BuildKit) can arrive in bursts of a few seconds. BuildKit's plain output only reprints a layer download after 5% or 5 seconds, and when several build steps run in parallel it holds back the output of the ones it isn't showing for up to 10 seconds. These delays are part of BuildKit, not checkship.
- **Waiting for health checks** prints `Waiting` then `Healthy` for each container, with nothing in between: a slow health check looks quiet but the deployment is still running.

The log is also available whole at `GET /apps/:id/deployments/:number/logs`, and streamed at `GET /apps/:id/deployments/:number/logs/stream` (see the [API](/reference/api#deployment-logs)).
