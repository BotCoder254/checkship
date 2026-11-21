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

## Custom domains {#custom-domains}

::: tip Step by step
[Guide › Custom domains](/guide/custom-domains) walks through it with an example, and [Guide › Cloudflare](/guide/cloudflare) covers domains at Cloudflare, including **Connect Cloudflare**, which sets up the DNS records (and a tunnel) for you.
:::

An environment can also be served at your own domains, such as `shop.acme.com`, next to the default address checkship gives it (`shop.apps.acme.com`). The default address keeps working: it's a fallback while DNS or the certificate isn't ready.

Domains are set **per environment and per service**, in **Settings › Environments › Domains**. A service's domains go to its first `http` port, the one that gets the default address. The first domain of the main service is the **primary**: the Visit button, the environment's link and the services list use it first.

1. **Add the domain** under the service that should answer it. Type the name only (`api.acme.com`): no `https://`, port or path. Internationalized names (`bücher.example`) are stored in their ASCII form (`xn--bcher-kva.example`) and shown in both forms.
2. **Point its DNS at the target**, at your DNS provider. The drawer lists the record to create:
   - a **CNAME** to the service's default address for a subdomain (`shop.acme.com CNAME shop.apps.acme.com`);
   - **A / AAAA** records with the target's addresses for a bare domain (`acme.com`), which can't have a CNAME.
3. **Save.** The environment is redeployed with the domain. On an `https` target, Let's Encrypt issues its certificate on the first visit.

::: tip Point DNS before saving
Let's Encrypt allows only **5 failed validations per domain per hour**, with no way around it. A domain saved before its DNS points at the target fails its certificate request until DNS is fixed, so check the status first.
:::

### Statuses

The drawer checks each saved domain (**Check again** refreshes it):

| Status | What it means |
| --- | --- |
| Points here | DNS sends visitors to this target: an address in common with the default address, or a CNAME to it |
| Points elsewhere | DNS sends them to other addresses (shown). Fix the record |
| No DNS record | The name doesn't exist yet |
| Behind Cloudflare | Every address is Cloudflare's: the record is proxied, so the check can't see past it (see below) |
| CAA blocks Let's Encrypt | A CAA record on the domain (or a parent) only allows other certificate authorities. Add `CAA 0 issue "letsencrypt.org"` |
| Default certificate | HTTPS isn't issued yet: the proxy answers with its own certificate. Visit the domain once, then check again |
| Certificate OK | A valid certificate, with its issuer and expiry date |

The check only resolves DNS (A, AAAA, CNAME and CAA records) and, for a domain pointing at the target, opens a TLS handshake to port 443 of the **target's own address** to read its certificate. It never connects anywhere else and sends no HTTP request.

### Behind Cloudflare

When a domain's nameservers are Cloudflare's, the drawer offers **Set it up with Cloudflare** ([Connect Cloudflare](/guide/cloudflare#connect)). Proxied records (orange cloud) work: certificates are requested with the HTTP-01 challenge, which Cloudflare forwards. Set **SSL/TLS** to **Full (strict)** once the certificate is issued (Full until then). A DNS-only record (grey cloud) works too.

### Rules

- A domain serves **one application**: checkship refuses a domain another application already has, and names it.
- Names inside the target's own domain (`*.apps.acme.com`) are refused: checkship already serves them. So is checkship's own address (`EXPOSED_ON`).
- At most **20 domains per service** and **50 per environment**.
- A target with a [manual proxy](/reference/targets#proxy) can't serve custom domains: checkship doesn't route its traffic.
- Wildcard domains (`*.acme.com`) aren't supported yet: they need a DNS-01 challenge with your DNS provider's API.
- A domain whose service doesn't exist or has no `http` port is ignored when deploying, and the deployment log says so.

### Removing a domain

Remove it from the drawer and save: the environment is redeployed without it, and the name is free for another application. **Delete its DNS record** at your provider too. A record left pointing at a server that no longer serves the name ("dangling DNS") could later be claimed by whatever answers there. The same goes for every domain of an application you delete.

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
