# Cloudflare

How to serve checkship applications at a domain whose DNS is at [Cloudflare](https://www.cloudflare.com/), such as `gitdesk.app`. Read [Custom domains](/guide/custom-domains) first for how domains work in checkship; this page covers what changes with Cloudflare.

The quickest way is **Connect Cloudflare**: you give checkship an API token once, and it sets up everything on Cloudflare, then keeps it in step with your domains. You can also [set it up by hand](#by-hand).

## Two ways visitors reach your target {#ways}

| | Cloudflare Tunnel | DNS record |
| --- | --- | --- |
| The target | Any machine, even a laptop or a server behind a home router | A server with a public IP |
| Open ports | None: the tunnel connects out to Cloudflare | 80 and 443 |
| HTTPS | Cloudflare's certificate, at Cloudflare | Let's Encrypt on the target (HTTP-01), plus Cloudflare's if proxied |
| Target URL in checkship | `http://…` (such as `http://docker.localhost`) | `https://…` |
| What checkship creates | A tunnel, a route per domain, a `cloudflared` container, a proxied CNAME per domain to the tunnel | A proxied CNAME per domain to the service's default address |

A **tunnel** runs `cloudflared` next to the target's proxy. It opens outgoing connections to Cloudflare, and Cloudflare sends your domain's visitors through them. Nothing listens on the internet, which is why it works from a desktop.

## Connect Cloudflare {#connect}

### Before you start {#connect-before}

- Add the domain to the application first: **Settings › Environments › Domains** (see [Custom domains](/guide/custom-domains#add)).
- The domain's zone must be **active** on Cloudflare: its nameservers are Cloudflare's. checkship notices this (it looks up the domain's nameservers) and only then shows **Set it up with Cloudflare** in the Domains drawer.

### 1. Create a token {#token}

In the Domains drawer, under **Set it up with Cloudflare**, press **Open Cloudflare**. Cloudflare's **Create API token** page opens with everything already chosen:

| Permission | Why checkship needs it |
| --- | --- |
| **Zone · Read** | Find your domains and the account they belong to |
| **DNS · Edit** | Create, update and delete the records it manages |
| **Cloudflare Tunnel · Edit** | Create the tunnel, set its routes, get its connector token. Newer Cloudflare pages may call it **Cloudflare One Connector: cloudflared · Write**. Only needed for a tunnel |

Click **Continue to summary**, then **Create Token**, and copy the token. Cloudflare shows it only once.

**User or account token.** Both work:
- the link opens **My Profile › API Tokens**, which makes a **user token** (it starts with `cfut_`);
- **Manage Account › Account API Tokens** makes an **account token** (`cfat_`), which keeps working if the person who created it leaves the account.

checkship checks a user token with Cloudflare's `/user/tokens/verify`, and an account token with `/accounts/<id>/tokens/verify` on its own account.

::: warning Keep the token secret
Paste it into checkship only, never into a chat, a ticket or a file. If it leaks, **Roll** it on the same Cloudflare page: the old one stops working at once.
:::

### 2. Connect {#paste}

Paste the token into the drawer and press **Connect**. checkship:
- checks it with Cloudflare (a malformed token is refused before anything is sent);
- lists the zones it can read and picks the account that holds your domains;
- shows **Connected to `<account>`** with the domains it can manage.

The token is stored on your checkship server, only ever sent to Cloudflare's API (`api.cloudflare.com`), and never returned by checkship's API or shown again. It is per **target**: every application on that target uses it. Only someone signed in to the dashboard can connect or disconnect, never an API key.

### 3. Choose the way, then set up {#setup}

Choose **Through a Cloudflare Tunnel** (marked "recommended here" for an `http://` target) or **With a DNS record**, then press **Set up and redeploy**. The domains are saved, and checkship does the rest. The first time, setup can take a minute or two, because it downloads the tunnel connector. Each step ticks off as it is done:

1. **Tunnel ready:** a tunnel named `checkship-<target id>` in your account (tunnel mode).
2. **Routes added:** one route per domain to the target's proxy, plus a final rule that answers 404 for anything else.
3. **DNS records set:** a proxied `CNAME` per domain, pointing at `<tunnel id>.cfargotunnel.com` (tunnel mode) or at the service's default address (DNS mode).
4. **Connector running:** a `cloudflared` container named `checkship-tunnel-<target id>` on the target, next to the proxy, restarted automatically. The tunnel's own token goes in through its environment, not its command line.

Then **https://your.domain is live** appears with a green checkmark. Cloudflare may take a minute to answer everywhere. From then on the domain shows **Behind Cloudflare** in the drawer, which is expected: the check can't see past Cloudflare's proxy.

### A record of yours is in the way {#conflict}

checkship only ever touches records it created. They carry the comment `managed by checkship (target <id>)`, visible in Cloudflare's DNS list. If the name already has an `A`, `AAAA` or `CNAME` record of yours, checkship leaves it alone and the drawer shows **Point `<domain>` to the tunnel**, in steps:

1. **Open DNS records of `<zone>`**, which opens the domain's **DNS › Records** page on Cloudflare.
2. **Delete the record in the way**: click **Edit** on its row, then **Delete**. An `AAAA` record to `100::` is Cloudflare's placeholder for "no server". If it serves a redirect rule or a Worker, keep it and use another domain instead.
3. **Add this record**: click **Add record** and fill it in as shown, with a Copy button for the target:

   | Field | Value |
   | --- | --- |
   | Type | `CNAME` |
   | Name | `@` for the domain itself (`gitdesk.app`), or the part before the zone for a subdomain (`api` for `api.gitdesk.app`) |
   | Target | `<tunnel id>.cfargotunnel.com` |
   | Proxy status | **Proxied** (orange cloud on): a tunnel only answers through Cloudflare's proxy |
   | TTL | Auto |

4. **Check the record**: checkship finds it, takes it over (adds its comment, if the token can edit DNS) and shows **`<domain>` points to the tunnel** with a green checkmark.

Or press **Replace the record**, and checkship deletes yours and adds its own. **Keep it** hides the steps and leaves the domain as it is.

### When the token is missing a permission {#permissions}

The drawer names the missing permission and quotes Cloudflare's answer (such as `10000: Authentication error`). You don't need a new token: on Cloudflare, **Edit** the token, add the permission and save. The token itself doesn't change. Then press **Set up and redeploy** again.

- **Without Cloudflare Tunnel · Edit:** connecting works, because listing tunnels only needs Read, but creating the tunnel is refused. The drawer switches to **With a DNS record** and tells you which permission to add for a tunnel.
- **Without DNS · Edit:** the tunnel side is still set up, and the drawer shows the record to add by hand, as in [A record of yours is in the way](#conflict). A record you add that way works as soon as it is proxied.

### Afterwards {#after}

- **It follows your domains.** Adding a domain to any application on the target, removing one, or deleting an application updates the routes and records by itself (a background job).
- **Domains in another Cloudflare account** than the token's are listed and left alone: set them up by hand, or connect a token of that account.
- **Target page › Cloudflare** shows the account, the token's permissions, the tunnel's state and connections, and the records checkship manages.
- **Disconnect** (on the target page) stops the connector, deletes the tunnel and the records checkship created, then forgets the token. Your other records are not touched. Revoke the token on Cloudflare too.
- Deleting the target disconnects it the same way.

## SSL/TLS settings {#ssl}

- **Tunnel:** Cloudflare serves HTTPS, and the tunnel talks HTTP to checkship's proxy on your machine. Turn on **SSL/TLS › Edge Certificates › Always Use HTTPS**, so `http://` visits are redirected: otherwise Cloudflare also serves your domain over plain HTTP. checkship builds its links from the target's URL (`http://…`), so the dashboard shows `http://your.domain`; with Always Use HTTPS on, those links land on HTTPS.
- **DNS record, proxied (orange cloud):** set **SSL/TLS › Overview** to **Full (strict)** once checkship has the certificate (**Full** until then). Never use **Flexible**: it would send your visitors' traffic unencrypted to the target.
- **DNS record, DNS only (grey cloud):** visitors reach the target directly and get its Let's Encrypt certificate.

checkship requests certificates with the **HTTP-01** challenge, which passes through Cloudflare's proxy. (The TLS-ALPN challenge can't, which is why checkship doesn't use it.)

## By hand {#by-hand}

Without a token, you can set up either way yourself.

### A tunnel {#tunnel}

1. Give the target an `http://` URL in checkship, add the domain to the application and deploy it.
2. In Cloudflare, open **Zero Trust › Networks › Tunnels**, **Create a tunnel**, choose **Cloudflared**, name it, and copy the token from the install command (the long string after `--token`).
3. In the tunnel, add a **Public hostname**: the domain (empty subdomain for the bare domain), service **HTTP**, URL the target's proxy as `cloudflared` sees it (`localhost:80` with host networking). Cloudflare creates the proxied DNS record.
4. Run the connector on the target:

   ```sh
   docker run -d --name cloudflared --restart unless-stopped --network host \
     cloudflare/cloudflared:2024.12.2 tunnel --no-autoupdate run --token <your tunnel token>
   ```

### A DNS record {#dns-record}

For a server with a public address and an `https://` target:

1. Add the domain in checkship; the drawer shows the record to create.
2. In Cloudflare, **DNS › Records › Add record**: a `CNAME` to the service's default address for a subdomain, or `A`/`AAAA` with the target's addresses for the bare domain (Cloudflare also flattens a CNAME there).
3. Proxied or DNS only both work; see [SSL/TLS settings](#ssl).
4. Save the domains in checkship. Let's Encrypt issues the certificate on the first visit.

## Troubleshooting {#troubleshooting}

| What you see | Why | Fix |
| --- | --- | --- |
| "Cloudflare refused this token" | Expired, revoked, limited to other IP addresses, or not copied whole | Check it on Cloudflare's token page, or create a new one |
| "The token is missing the Cloudflare Tunnel · Edit permission" (`10000: Authentication error`) | The token can read tunnels but not create them | Edit the token, add **Cloudflare Tunnel · Edit**, set up again |
| "The token is missing the DNS · Edit permission" | The token can only read DNS | Add **DNS · Edit**, or add the record by hand as the drawer shows |
| Setup seems slow the first time | The connector image is being downloaded | Wait a minute or two; the drawer says so |
| `https://your.domain` doesn't resolve on one computer | Its DNS cache remembers a moment the name had no record | Wait, or flush the cache (`resolvectl flush-caches` on Linux) |
| Cloudflare error **1033** | The tunnel has no running connector | Check `checkship-tunnel-<target>` is running on the target; **Set up and redeploy** starts it again |
| Cloudflare error **502** or **404** through the tunnel | The route or the application isn't there | Check the deployment succeeded and the domain is on the right service |
| Cloudflare error **525/526** (DNS mode) | SSL mode Full (strict) before the certificate was issued | Use **Full** until **Certificate OK** shows, then **Full (strict)** |
| "Not in this Cloudflare account" | The domain's zone is in another account | Connect a token of that account, or set it up by hand |

## Why a token, not a "Sign in with Cloudflare" button {#why-token}

As of November 2026, Cloudflare offers two other ways, which a self-hosted tool can't use on its own:

- **Domain Connect** (Cloudflare's consent screen, then the records are applied): only for providers whose signed template is in the [Domain Connect templates](https://github.com/Domain-Connect/Templates) and onboarded by Cloudflare. Every request must be signed by the provider, so every checkship instance would depend on a central signing service.
- **Cloudflare OAuth** (authorization code with PKCE): each self-hosted instance has its own address, so each would need its own OAuth client, or go through a central relay.

The pre-filled token link works for everyone today, with nothing in between your server and Cloudflare.
