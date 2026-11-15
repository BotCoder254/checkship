import { expect, test, type Page } from '@playwright/test';

/**
 * Application pages, drawers, drop zones and highlighted code (DESIGN.md §13.6) against the real Go
 * API. Creates its own target and application, so it runs on its own. Screenshots go to each
 * test's output folder (test-results/) for review.
 */

const ADMIN = { email: 'admin@example.com', password: 'admin' };

const COMPOSE = `# e2e
services:
  app:
    image: nginx:1.27-alpine
    ports:
      - "8080:80"
`;

async function signIn(page: Page, path = '/') {
	await page.goto(`/signin/?redirectTo=${encodeURIComponent(path)}`);
	await page.getByLabel('Email').fill(ADMIN.email);
	await page.getByLabel('Password').fill(ADMIN.password);
	await page.getByRole('button', { name: 'Sign in' }).click();
	await expect(page).toHaveURL(new RegExp(`${path.replaceAll('/', '\\/')}$`));
}

async function shot(page: Page, name: string) {
	await page.screenshot({ path: test.info().outputPath(`${name}.png`) });
}

/**
 * A target and an application of our own. The target is a remote one (its configuration fails
 * here, which doesn't matter): only one target may use the local Docker engine, and shell.test.ts
 * creates that one.
 */
async function fixtures(page: Page) {
	const targets = await (await page.request.get('/api/v1/targets')).json();
	let target = targets.find((t: { name: string }) => t.name === 'pages-target')?.id;
	if (!target) {
		const created = await page.request.post('/api/v1/targets', {
			data: { name: 'pages-target', docker: { host: 'pages.e2e.invalid', user: 'e2e' } }
		});
		expect(created.ok()).toBe(true);
		target = (await created.json()).id;
	}

	const apps = await (await page.request.get('/api/v1/apps')).json();
	let app = apps.find((a: { name: string }) => a.name === 'pages-app')?.id;
	if (!app) {
		const created = await page.request.post('/api/v1/apps', {
			data: { name: 'pages-app', production: { target }, staging: { target } }
		});
		expect(created.ok()).toBe(true);
		app = (await created.json()).id;
	}
	return { target, app };
}

function watchErrors(page: Page) {
	const errors: string[] = [];
	page.on('pageerror', (error) => errors.push(error.message));
	page.on('console', (message) => {
		if (message.type() === 'error' && !message.text().startsWith('Failed to load resource')) {
			errors.push(message.text());
		}
	});
	return errors;
}

test.describe('desktop', () => {
	test.use({ viewport: { width: 1440, height: 900 } });

	test('new deployment drawer: compose drop zone, highlighted editor, then the preview', async ({
		page
	}) => {
		const errors = watchErrors(page);
		await signIn(page);
		const { app } = await fixtures(page);

		await page.goto(`/apps/${app}/`);
		await expect(page.getByRole('heading', { level: 1, name: 'pages-app' })).toBeVisible();
		// Tabs, the staging → production cards, and nothing deployed yet.
		await expect(page.locator('main').getByRole('link', { name: 'Overview' })).toHaveAttribute(
			'aria-current',
			'page'
		);
		await expect(page.getByRole('heading', { level: 3, name: 'Staging' })).toBeVisible();
		await expect(page.getByRole('heading', { level: 3, name: 'Production' })).toBeVisible();
		await expect(page.getByText('Deploy to staging first, then promote it here')).toBeVisible();
		// Stacked: production under staging, both the full width and the same height.
		const stagingCard = (await page
			.locator('section[aria-labelledby="card-staging"]')
			.boundingBox())!;
		const productionCard = (await page
			.locator('section[aria-labelledby="card-production"]')
			.boundingBox())!;
		expect(productionCard.y).toBeGreaterThan(stagingCard.y + stagingCard.height);
		expect(Math.round(productionCard.width)).toBe(Math.round(stagingCard.width));
		expect(Math.round(productionCard.height)).toBe(Math.round(stagingCard.height));
		await shot(page, 'overview-empty');

		// No Git on this application: one-click Deploy opens the drawer instead.
		await page.getByRole('button', { name: 'Deploy', exact: true }).click();

		// A floating drawer on the right, 560px, the page still behind it.
		const drawer = page.getByRole('dialog', { name: 'New deployment' });
		await expect(drawer).toBeVisible();
		await expect.poll(async () => Math.round((await drawer.boundingBox())!.width)).toBe(560);
		await expect
			.poll(async () => {
				const box = (await drawer.boundingBox())!;
				return Math.round(box.x + box.width);
			})
			.toBe(1440 - 8);
		await expect(drawer).toHaveCSS('border-radius', '10px');

		// No Git on this application: compose file is the default source.
		await expect(drawer.getByRole('radio', { name: 'Compose file' })).toHaveAttribute(
			'aria-checked',
			'true'
		);
		await expect(drawer.getByRole('radio', { name: 'Git' })).toHaveCount(0);

		// Dropping (here: choosing) a compose.yml fills the editor.
		await drawer.locator('input[type="file"]').setInputFiles({
			name: 'compose.yml',
			mimeType: 'text/yaml',
			buffer: Buffer.from(COMPOSE)
		});
		await expect(drawer.getByText('compose.yml', { exact: true })).toBeVisible();
		await expect(drawer.getByLabel('Compose file')).toHaveValue(COMPOSE);

		// Highlighted with palette tokens: keys in accent.
		const key = drawer.locator('pre span', { hasText: /^services$/ }).first();
		await expect(key).toHaveAttribute('style', /--shiki-token-keyword/);
		await expect(key).toHaveCSS('color', 'rgb(127, 149, 255)');

		// The cURL command is highlighted too.
		await drawer.getByRole('button', { name: 'Deploy this from CI with cURL' }).click();
		await expect(drawer.locator('pre').filter({ hasText: 'curl -i -X POST' })).toBeVisible();
		await shot(page, 'deploy-drawer-compose');

		await drawer.getByRole('button', { name: 'Deploy', exact: true }).click();
		const preview = page.getByRole('dialog', { name: 'Deployment #1' });
		await expect(preview).toBeVisible();
		await expect.poll(async () => Math.round((await preview.boundingBox())!.width)).toBe(680);
		await expect(preview.getByRole('link', { name: 'Open full page' })).toBeVisible();
		await shot(page, 'deployment-preview');

		await page.keyboard.press('Escape');
		await expect(preview).toHaveCount(0);
		expect(errors).toEqual([]);
	});

	test('archive source: a wrong file is refused in its row', async ({ page }) => {
		await signIn(page);
		const { app } = await fixtures(page);
		await page.goto(`/apps/${app}/deployments/new/`);
		await expect(page.getByRole('heading', { level: 1, name: 'New deployment' })).toBeVisible();

		await page.getByRole('radio', { name: 'Archive' }).click();
		const zone = page.getByRole('button', { name: /Add a \.tar\.gz archive/ });
		await expect(zone).toBeVisible();
		await page.locator('input[type="file"]').setInputFiles({
			name: 'notes.pdf',
			mimeType: 'application/pdf',
			buffer: Buffer.from('%PDF')
		});
		await expect(
			page.getByRole('alert').filter({ hasText: 'isn’t a .tar.gz archive' })
		).toBeVisible();
		await shot(page, 'deploy-archive-invalid');

		await page.getByRole('button', { name: 'Remove the file' }).click();
		await page.locator('input[type="file"]').setInputFiles({
			name: 'app.tar.gz',
			mimeType: 'application/gzip',
			buffer: Buffer.from([0x1f, 0x8b])
		});
		await expect(page.getByText('app.tar.gz')).toBeVisible();
	});

	test('settings: two columns, variables saved from their drawer, delete asks for the name', async ({
		page
	}) => {
		await signIn(page);
		const { app } = await fixtures(page);
		await page.goto(`/apps/${app}/edit/`);
		await expect(page.getByRole('heading', { level: 1, name: 'Settings' })).toBeVisible();

		// Section title on the left, fields on the right.
		const general = page.getByRole('heading', { level: 2, name: 'General' });
		const name = page.getByLabel('Name', { exact: true });
		expect((await name.boundingBox())!.x).toBeGreaterThan((await general.boundingBox())!.x + 260);

		await page.getByRole('button', { name: 'Variables' }).first().click();
		const drawer = page.getByRole('dialog', { name: 'Production variables' });
		await expect(drawer).toBeVisible();
		await drawer.getByRole('button', { name: 'Add service' }).click();
		await drawer.getByRole('textbox', { name: 'Service', exact: true }).fill('app');
		await drawer.getByRole('textbox', { name: 'Variables', exact: true }).fill('LOG_LEVEL=debug');
		await expect(drawer.locator('pre span', { hasText: 'LOG_LEVEL' }).first()).toBeVisible();
		await shot(page, 'variables-drawer');
		await drawer.getByRole('button', { name: 'Save variables' }).click();
		await expect(drawer).toHaveCount(0);
		await expect(page.getByText('1 service · 1 variable')).toBeVisible();

		const saved = await (await page.request.get(`/api/v1/apps/${app}`)).json();
		expect(saved.production.vars).toEqual({ app: { LOG_LEVEL: 'debug' } });
		await shot(page, 'settings');

		await page.getByRole('button', { name: 'Delete application' }).click();
		const dialog = page.getByRole('alertdialog');
		const confirm = dialog.getByRole('button', { name: 'Delete application' });
		await expect(confirm).toBeDisabled();
		await dialog.getByLabel('Type pages-app to confirm').fill('pages-app');
		await expect(confirm).toBeEnabled();
		await dialog.getByRole('button', { name: 'Cancel' }).click();
	});

	test('registries: create, edit and delete from the drawer', async ({ page }) => {
		await signIn(page, '/registries/');
		await page.getByRole('button', { name: 'New registry' }).first().click();
		const drawer = page.getByRole('dialog', { name: 'New registry' });
		// A preset fills the URL and names the token to create; its mark is drawn.
		await drawer.getByRole('radio', { name: 'GitHub' }).click();
		await expect(drawer.getByLabel('Url')).toHaveValue('https://ghcr.io');
		await expect(drawer.getByRole('radio', { name: 'GitHub' }).locator('svg')).toHaveCount(1);
		await expect(drawer.getByText('read:packages').first()).toBeVisible();
		await drawer.getByLabel('Name', { exact: true }).fill('e2e registry');
		await drawer.getByLabel('Username').fill('e2e-bot');
		await drawer.getByLabel('Password or token').fill('not-a-real-token');
		await shot(page, 'registry-drawer');
		await drawer.getByRole('button', { name: 'Create registry' }).click();
		await expect(drawer).toHaveCount(0);

		const row = page.getByRole('row').filter({ hasText: 'e2e registry' });
		await expect(row).toBeVisible();
		await expect(row).toContainText('e2e-bot');
		await row.click();
		const edit = page.getByRole('dialog', { name: 'e2e registry' });
		await expect(edit.getByLabel('Url')).toHaveValue('https://ghcr.io');
		await expect(edit.getByText('GitHub, recognised from its URL')).toBeVisible();
		// The saved token is masked by the server, never filled back in.
		await expect(edit.getByLabel('Password or token')).toHaveValue('');
		await edit.getByRole('button', { name: 'Delete registry' }).click();
		const dialog = page.getByRole('alertdialog');
		await dialog.getByLabel('Type e2e registry to confirm').fill('e2e registry');
		await dialog.getByRole('button', { name: 'Delete registry' }).click();
		await expect(row).toHaveCount(0);
	});

	test('targets: overview, failed state, settings and delete blocked while in use', async ({
		page
	}) => {
		await signIn(page, '/targets/');
		const { target } = await fixtures(page);
		const row = page.getByRole('row').filter({ hasText: 'pages-target' });
		await expect(row).toContainText('e2e@pages.e2e.invalid:22');

		// It can't reach that host: wait for the configuration to fail.
		await expect
			.poll(
				async () =>
					(await (await page.request.get(`/api/v1/targets/${target}`)).json()).state.status,
				{ timeout: 30_000 }
			)
			.toBe(1);
		await page.goto(`/targets/${target}/`);
		await expect(page.getByRole('heading', { level: 1, name: 'pages-target' })).toBeVisible();
		await expect(page.getByRole('alert').filter({ hasText: 'couldn’t configure' })).toBeVisible();
		await expect(page.getByText('pages.e2e.invalid on port 22')).toBeVisible();
		await expect(page.getByRole('link', { name: /pages-app/ })).toBeVisible();
		await shot(page, 'target-failed');

		await page.getByRole('link', { name: 'Settings' }).last().click();
		await expect(page).toHaveURL(new RegExp(`/targets/${target}/edit/$`));
		await expect(page.getByLabel('Host')).toHaveAttribute('readonly', '');
		await expect(page.getByLabel('User')).toHaveAttribute('placeholder', 'docker');

		await page.getByRole('button', { name: 'Delete target' }).click();
		const dialog = page.getByRole('alertdialog');
		await expect(dialog).toContainText('pages-target is still in use');
		await expect(dialog.getByRole('link', { name: /pages-app/ }).first()).toBeVisible();
		await expect(dialog.getByRole('button', { name: 'Delete target' })).toBeDisabled();
		await shot(page, 'target-delete-blocked');
		await dialog.getByRole('button', { name: 'Close' }).click();
	});

	test('jobs: live queue, state filter, details in a drawer', async ({ page }) => {
		await signIn(page, '/jobs/');
		await expect(page.getByRole('heading', { level: 1, name: 'Jobs' })).toBeVisible();
		await expect(page.getByText('Live · updates every 5 s')).toBeVisible();
		await page.getByRole('button', { name: 'Pause updates' }).click();
		await expect(page.getByText('Updates paused')).toBeVisible();
		await page.getByRole('button', { name: 'Resume updates' }).click();

		// The API filters by state.
		for (const status of ['running', 'failed', 'pending']) {
			const res = await page.request.get(`/api/v1/jobs?status=${status}`);
			expect(res.ok()).toBe(true);
			for (const job of (await res.json()).data as { retrieved: boolean; error_code?: string }[]) {
				expect(job.retrieved).toBe(status === 'running');
				if (status !== 'running') expect(!!job.error_code).toBe(status === 'failed');
			}
		}

		// The State filter is a menu whose options each carry an icon.
		const state = page.getByRole('button', { name: /^State:/ });
		await state.click();
		const failed = page.getByRole('menuitemradio', { name: 'Failed, retrying' });
		await expect(failed.locator('svg').first()).toBeVisible();
		await failed.click();
		await expect(state).toContainText('Failed, retrying');
		await state.click();
		await page.getByRole('menuitemradio', { name: 'Any state' }).click();

		const rows = page.getByRole('row').filter({ has: page.getByRole('button') });
		if ((await rows.count()) === 0) {
			await expect(page.getByText('The queue is empty')).toBeVisible();
			return;
		}
		await rows.first().getByRole('button').first().click();
		const drawer = page.getByRole('dialog');
		await expect(drawer.getByRole('heading', { name: 'Payload' })).toBeVisible();
		await expect(drawer.getByRole('heading', { name: 'Policy' })).toBeVisible();
		await shot(page, 'job-drawer');
	});

	test('deployments tab: server-side filters and a menu on each row', async ({ page }) => {
		await signIn(page);
		const { app } = await fixtures(page);
		// The first test queued #1 on production; make sure there is one either way.
		const existing = await (await page.request.get(`/api/v1/apps/${app}/deployments`)).json();
		if (existing.total === 0) {
			await page.request.post(`/api/v1/apps/${app}/deployments`, {
				data: { environment: 'production', raw: COMPOSE }
			});
		}

		// The API filters: by number, and by a status no deployment has.
		const byNumber = await (
			await page.request.get(`/api/v1/apps/${app}/deployments?search=%231`)
		).json();
		expect(byNumber.data.map((d: { deployment_number: number }) => d.deployment_number)).toEqual([
			1
		]);
		const none = await (
			await page.request.get(`/api/v1/apps/${app}/deployments?status=3&search=nope`)
		).json();
		expect(none.total).toBe(0);

		await page.goto(`/apps/${app}/deployments/`);
		await expect(page.getByRole('link', { name: /Deployments/ })).toHaveAttribute(
			'aria-current',
			'page'
		);
		const row = page.getByRole('row').filter({ hasText: '#1' }).first();
		await expect(row).toBeVisible();

		await row.getByRole('button', { name: 'Actions for deployment #1' }).click();
		await expect(page.getByRole('menuitem', { name: 'View logs' })).toBeVisible();
		await expect(page.getByRole('menuitem', { name: 'Copy link' })).toBeVisible();
		// Every menu item has its icon.
		for (const item of await page.getByRole('menuitem').all()) {
			await expect(item.locator('svg')).toHaveCount(1);
		}
		await shot(page, 'deployments-menu');
		await page.keyboard.press('Escape');

		await page.getByRole('searchbox', { name: 'Search' }).fill('no-such-branch');
		await expect(page.getByText('No deployment matches')).toBeVisible();
		await page.getByRole('searchbox', { name: 'Search' }).fill('');
		await expect(row).toBeVisible();
		await shot(page, 'deployments-tab');
	});

	test('overview: bento tiles, filters, live activity and paging', async ({ page, request }) => {
		const errors = watchErrors(page);
		await signIn(page);
		const { app } = await fixtures(page);

		// The API is session-only: the CI key is refused, and bad values fall back.
		const key = (await (await page.request.get('/api/v1/profile')).json()).api_key as string;
		const withKey = await request.get('/api/v1/dashboard/summary', {
			headers: { Authorization: `Bearer ${key}` }
		});
		expect(withKey.status()).toBe(401);
		const fallback = await page.request.get('/api/v1/dashboard/summary?range=365&tz=99999');
		expect((await fallback.json()).days).toBe(30);
		expect(fallback.headers()['cache-control']).toBe('private, no-store');
		expect((await page.request.get('/api/v1/dashboard/activity?cursor=bad')).status()).toBe(400);

		await page.goto('/');
		await expect(page.getByRole('heading', { level: 1, name: 'Overview' })).toBeVisible();
		// The key figures are one strip (a region), not four cards; each figure is a named group.
		const strip = page.getByRole('region', { name: /Key figures, last 30 days/ });
		for (const title of ['Deployments', 'Success rate', 'Median duration', 'Time to recover']) {
			await expect(strip.getByRole('group', { name: title })).toBeVisible();
		}
		await expect(page.getByText('Live', { exact: true }).first()).toBeVisible();

		// Changing the period asks the server for it.
		const summary = page.waitForRequest(
			(r) => r.url().includes('/dashboard/summary?') && r.url().includes('range=7')
		);
		await page.getByRole('combobox', { name: 'Period' }).click();
		await page.getByRole('option', { name: 'Last 7 days' }).click();
		await summary;

		// A deployment queued elsewhere shows up without reloading the page.
		const activity = page.getByRole('region', { name: 'Activity' });
		const queued = await page.request.post(`/api/v1/apps/${app}/deployments`, {
			data: { environment: 'production', raw: COMPOSE }
		});
		expect(queued.ok()).toBe(true);
		const { deployment_number: number } = await queued.json();
		await expect(activity.getByRole('link', { name: `pages-app #${number}` })).toBeVisible({
			timeout: 10_000
		});

		// Load more appends older rows without repeating any.
		const loadMore = activity.getByRole('button', { name: 'Load more' });
		if (await loadMore.isVisible()) {
			await loadMore.click();
			await expect(loadMore).toBeEnabled();
			const names = await activity.getByRole('link').allTextContents();
			expect(new Set(names).size).toBe(names.length);
		}

		// Only the workspace scrolls: nothing (such as the chart's screen-reader table) may stretch
		// the document past the window, which showed the page ground under the shell.
		expect(
			await page.evaluate(() => document.documentElement.scrollHeight - window.innerHeight)
		).toBeLessThanOrEqual(0);

		await shot(page, 'overview');
		expect(errors).toEqual([]);
	});

	test('every page renders without errors; unknown ids show not found', async ({ page }) => {
		const errors = watchErrors(page);
		await signIn(page);
		const { app } = await fixtures(page);

		for (const path of [
			'/',
			'/apps/',
			'/jobs/',
			'/apps/new/',
			'/registries/new/',
			`/apps/${app}/`,
			`/apps/${app}/edit/`,
			`/apps/${app}/deployments/new/`,
			`/apps/${app}/deployments/`,
			'/targets/new/'
		]) {
			await page.goto(path);
			await expect(page.locator('main h1')).toBeVisible();
			await expect(page.getByText('not found', { exact: false })).toHaveCount(0);
		}

		await page.goto('/apps/does-not-exist/');
		await expect(page.getByRole('heading', { name: 'Application not found' })).toBeVisible();
		await expect(page.getByRole('link', { name: 'Back to applications' })).toBeVisible();
		await shot(page, 'not-found');
		expect(errors).toEqual([]);
	});
});

test('phone 390: the drawer fills the screen, 44px controls', async ({ page }) => {
	await page.setViewportSize({ width: 390, height: 844 });
	await signIn(page);
	const { app } = await fixtures(page);
	await page.goto(`/apps/${app}/`);
	await page.getByRole('button', { name: 'Deploy', exact: true }).click();
	const drawer = page.getByRole('dialog', { name: 'New deployment' });
	await expect.poll(async () => Math.round((await drawer.boundingBox())!.width)).toBe(390);
	const radio = drawer.getByRole('radio', { name: 'Compose file' });
	expect(Math.round((await radio.boundingBox())!.height)).toBe(36);
	const deploy = drawer.getByRole('button', { name: 'Deploy', exact: true });
	expect(Math.round((await deploy.boundingBox())!.height)).toBe(44);
	await shot(page, 'phone-drawer');
});
