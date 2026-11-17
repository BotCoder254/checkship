/** The documentation site; empty states link to the page that explains what to add. */
const BASE = 'https://checkship-deploy.github.io/checkship';

export default {
	quickstart: `${BASE}/guide/quickstart.html`,
	applications: `${BASE}/reference/applications.html`,
	targets: `${BASE}/reference/targets.html`,
	registries: `${BASE}/reference/registries.html`,
	backups: `${BASE}/reference/backups.html`,
	deployments: `${BASE}/reference/deployments.html`
} as const;
