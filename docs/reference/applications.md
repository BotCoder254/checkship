# Applications

Defines a new **stack of services** exposed on a [target subdomain](/reference/targets). The name will be used as a subdomain and should be unique **per target and environment**.

## Multiple exposed services

When you define a complete stack for your application, you may have multiple services exposed. This is totally allowed by **checkship**. When [identifying services](/reference/providers/docker#exposing-services) which must be exposed, the first one in **alphabetical order** will become the **default service** and take the default subdomain.

Other services will be exposed using a subdomain on the default one.

## Environments {#environments}

Only 2 environments are managed by **checkship** at the moment: **production** and **staging**.

::: info
Any updates on an application environment will trigger a redeploy of the latest deployment.
:::

Each one of those environment can be deployed on a separate target and configured appropriately with specific environment variables.

::: warning
When you update an environment target, if at least one successful deployment have been made on the old target, a cleanup job will be queued, meaning all resources for this specific application and environment will be **deleted** on the old target.

This prevent a target from having dangling applications.
:::

### Production

Represents the main environment. The **default service** will be exposed on `<target scheme>://<app name>.<target root url>`. Any additional exposed services will add another level such as `<target scheme>://<service name>.<app name>.<target root url>`.

### Staging

For the staging environment, a `-staging` suffix is added to the application name: `<target scheme>://<app name>-staging.<target root url>`.

## Logs {#logs}

The **Logs** tab of an application shows what its containers write, live. Choose the environment, then follow every service at once or only one. On the overview, **App logs** opens it on an environment that is live, and **Deployment log** still shows what a deployment printed while it ran.

- Lines come from Docker on the target. **checkship** reads them while the tab is open and stores nothing.
- The tab loads the last 200 lines of each container (100 to 5,000 with the lines menu), in time order, then follows new ones.
- Redeploys, restarts and new services join the stream on their own. If every container stops, the stream stays open and picks up the next start.
- Lines written to standard error carry a `stderr` marker. Lines that look like errors are shown in red, and **Errors only** keeps only those.
- The search narrows the lines shown. Start a word with `-` to hide the lines that contain it, for example `timeout -healthz`.
- **Pause** freezes the view while lines keep arriving, and **Follow** goes back to the newest.
- Each line has its own copy button. **Copy** copies the lines shown, **Copy as one line** joins them with `|` to paste in a chat or an issue, and **Download .log** saves them to a file.
- Above the lines, each container shows its state, how long it has been up, how many times it restarted, its health check and its last exit code.

Colour codes written by applications are removed, and the view keeps the newest 5,000 lines.

::: info Logging drivers
**checkship** reads logs the way `docker logs` does. That works with the `json-file`, `local` and `journald` logging drivers, and with other drivers while Docker's dual logging is on (the default since Docker Engine 20.10). When the logs can't be read, the tab says so.
:::

::: tip Behind a reverse proxy
Logs are sent as server-sent events (`text/event-stream`). If a proxy in front of **checkship** compresses or buffers responses, leave that type out, or lines arrive in bursts.
:::

Each user can follow 3 log streams at once, and 20 can be open in all, because each one keeps connections open to the target. The same logs are available from the [API](/reference/api#runtime-logs).

## Cleanup

Deleting an application will (if at least one deployment has been successful on a target) remove **everything created by checkship** on it:

- Containers
- Images
- Networks
- Volumes

Unless you turn off **Back up volumes first** in the deletion dialog, each environment's volumes are [backed up](/reference/backups#before-delete) before they are removed, and nothing is deleted if a backup fails. The backups stay under **Backups** after the application is gone.

Changing the target of an environment cleans up the old target the same way, after [moving its volume data](/reference/backups#move) to the new one if you keep that option on.

::: info
If you want to delete an application and the cleanup could not be done correctly because of a particular situation, you can [cancel the cleanup job](/reference/jobs#cancellation) from the **jobs** page.
:::
