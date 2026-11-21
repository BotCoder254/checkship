# Custom domains

Serve an application at your own domain, such as `app.acme.com`, next to the address checkship gives it. This page walks through it from start to finish. The rules and every status are in the reference: [Applications › Custom domains](/reference/applications#custom-domains). If your domain's DNS is at Cloudflare, [Cloudflare](/guide/cloudflare) can do the DNS part for you.

## How it works {#how-it-works}

Every environment already has a **default address** on its target: `<app>.<target domain>` (and `<service>.<app>.<target domain>` for other services), such as `notes-web.apps.acme.com`. Your **custom domains** are added next to it:

```
visitor ──▶ app.acme.com ──DNS──▶ your target's proxy (Traefik) ──▶ the service's container
visitor ──▶ notes-web.apps.acme.com ──────────────────────────────▶ the same container
```

- **Per environment, per service.** Production and staging have their own domains. In each, a domain belongs to one service of your compose file and reaches its first web port, the one that gets the default address.
- **The default address keeps working.** It's a fallback while DNS or the certificate isn't ready, and stays useful for testing.
- **Primary domain.** The first domain of the main service becomes the environment's primary address: the **Visit** button, the overview and the services list show it first, with the default address "also at".
- **One router per domain.** At deploy time checkship gives each domain its own route on the target's proxy, so each gets its own certificate, and a domain whose DNS isn't ready can't break the others.

## Before you start {#before}

- **A target with an automatic proxy**, meaning a target with a URL (`http://…` or `https://…`). A target with a [manual proxy](/reference/targets#proxy) can't serve custom domains: you run your own proxy there.
- **A domain you control**, at any DNS provider.
- **Where the target is reachable from:**
  - An `https://` target on a server with a public address: visitors come straight to it, and checkship gets a Let's Encrypt certificate for each domain (ports **80** and **443** must be open).
  - An `http://` target behind a home router or on a laptop: the internet can't reach it, so go through a [Cloudflare Tunnel](/guide/cloudflare#ways). Cloudflare handles HTTPS.

## Step by step {#steps}

### 1. Add the domain {#add}

Open the application, then **Settings › Environments**, and press **Domains** on the environment's row. The drawer explains each part as you go.

- Under the service that should answer (the website, or the API), type the name only: `app.acme.com`. No `https://`, port or path.
- Add as many as you need (up to 20 per service). Use **Another service** for a service that hasn't been deployed yet.
- Internationalized names (`bücher.example`) are accepted, stored in their ASCII form (`xn--bcher-kva.example`) and shown in both.

The drawer refuses, with the reason, a name another application already uses (and names it), a name inside the target's own domain (checkship already serves those), checkship's own address, wildcards, IP addresses and anything that isn't a hostname.

### 2. Point DNS at the target {#dns}

The drawer's **DNS records to create** lists the exact record for each domain:

| Your domain | Record | Value |
| --- | --- | --- |
| A subdomain, `app.acme.com` | `CNAME` | the service's default address, `notes-web.apps.acme.com` |
| The bare domain, `acme.com` | `A` (and `AAAA`) | the target's addresses, `203.0.113.10` |

A bare domain can't have a CNAME in standard DNS, which is why it gets the addresses instead. (Cloudflare accepts a CNAME at the root and flattens it.)

Create the record at your DNS provider, then press **Check again** in the drawer. Wait until the domain shows **Points here** (or **Behind Cloudflare**).

::: tip Point DNS before saving
On an `https` target, Let's Encrypt allows only **5 failed validations per domain per hour**. A domain saved before its DNS points at the target fails its certificate until DNS is fixed, so check the status first.
:::

### 3. Save and redeploy {#save}

Press **Save and redeploy**. checkship saves the domains and redeploys the environment with a route per domain. The deployment log says what it did:

```
[INFO] serving web on app.acme.com
```

Domains it can't serve are listed as warnings in the log, such as a service that isn't in the compose file or has no web port.

### 4. Check it {#check}

- Open the domain in a browser. On an `https` target, the certificate is issued on the first visit, within a few seconds.
- In the drawer, **Check again** should now show **Points here** and **Certificate OK**.
- The overview shows the domain as the environment's address, with the default one under it.

## What the statuses mean {#statuses}

| Status | What it means | What to do |
| --- | --- | --- |
| Points here | DNS sends visitors to this target | Nothing |
| Behind Cloudflare | Every address is Cloudflare's: the record is proxied | Nothing; the check can't see past Cloudflare |
| Points elsewhere | DNS sends visitors to other addresses (shown) | Fix the record with the value in **DNS records to create** |
| No DNS record | The name doesn't exist yet | Create the record |
| CAA blocks Let's Encrypt | A CAA record only allows other certificate authorities | Add `CAA 0 issue "letsencrypt.org"` |
| Default certificate | HTTPS isn't issued yet | Visit the domain once, then check again |
| Certificate OK | A valid certificate, with its issuer and expiry | Nothing: it renews on its own |
| No answer on 443 | Nothing answered on port 443 of the target | Open port 443 in the server's firewall |

The check only resolves DNS (A, AAAA, CNAME, CAA and NS records) and, for a domain that points at the target, opens a TLS handshake to port 443 of **the target's own address** to read the certificate. It never connects anywhere else.

## Example: a web app and its API on their own domains {#example}

A common setup: a web app at `app.acme.com` that calls its API at `api.acme.com`. These are two checkship applications on the same target. The checkship repository's tests used the same pair on `gitdesk.app` behind a Cloudflare Tunnel.

**The API** (`notes-api`), a compose file whose service `api` listens on port 8000:

```yaml
services:
  api:
    build: .
    restart: unless-stopped
    ports:
      - "8000:8000"
```

**The web app** (`notes-web`), whose service `web` is nginx serving a page:

```yaml
services:
  web:
    build: .
    restart: unless-stopped
    ports:
      - "8080:80"
```

1. Create both applications on the target. Add `api.acme.com` under `api` in `notes-api`'s production domains, and `app.acme.com` under `web` in `notes-web`'s.
2. Point both names at the target (two CNAMEs, or let [Connect Cloudflare](/guide/cloudflare) create them), then deploy both.
3. In the web app, call the API by its domain, not its default address, so it keeps working wherever the API moves:

   ```js
   const API = 'https://api.acme.com';
   const notes = await fetch(`${API}/api/notes`).then((res) => res.json());
   ```

4. The API must allow the web app's origin, since they are two domains (CORS):

   ```
   Access-Control-Allow-Origin: https://app.acme.com
   Access-Control-Allow-Methods: GET, POST, OPTIONS
   Access-Control-Allow-Headers: Content-Type
   ```

Both default addresses keep answering too: `notes-api.apps.acme.com` and `notes-web.apps.acme.com`.

## With the API {#api}

Everything in the drawer is in the [API](/reference/api#custom-domains). Domains are part of an environment, next to its variables:

```sh
curl -X PATCH -H "Authorization: Bearer <API key>" -H "Content-Type: application/json" \
  -d '{"production": {"target": "<target id>", "domains": {"web": ["app.acme.com"]}}}' \
  https://checkship.example.com/api/v1/apps/<app id>
```

Changing domains redeploys the environment. `GET /api/v1/apps/<app id>/domains/check?environment=production` returns each domain's status.

## Removing a domain {#remove}

Remove it in the drawer and save: the environment is redeployed without it, and the name is free for another application. Then **delete its DNS record** at your provider. A record left pointing at a server that no longer serves the name (dangling DNS) could later be claimed by whatever answers there. The same goes for every domain of an application you delete. With [Connect Cloudflare](/guide/cloudflare), checkship deletes the records it created for you.

## Troubleshooting {#troubleshooting}

| What you see | Why | Fix |
| --- | --- | --- |
| `domain_already_used` | Another application has the domain | Remove it there first; the error names the application |
| `domain_reserved_by_target` | The name is inside the target's domain | Use the default address, or another name |
| `custom_domains_need_automatic_proxy` | The target has a manual proxy | Give the target a URL, or route the domain in your own proxy |
| The browser shows a certificate warning | The certificate isn't issued yet (Default certificate) | Wait for DNS to point here, visit once, check again |
| 404 from the proxy | The redeploy is still running, or the service has no web port | Wait for the deployment, then read its log |
| It works from one device but not another | That device's DNS cache still has the old answer | Wait for the TTL, or flush the cache (`resolvectl flush-caches` on Linux) |
| Cloudflare error 1033 or 530 | The tunnel's connector isn't running | See [Cloudflare › Troubleshooting](/guide/cloudflare#troubleshooting) |

Not supported yet: wildcard domains (`*.acme.com`, they need a DNS-01 challenge with your DNS provider), path prefixes, and redirects between domains (such as `www` to the bare domain).
