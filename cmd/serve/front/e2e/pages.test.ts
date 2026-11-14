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
		await expect(page.getByRole('link', { name: 'Overview' })).toHaveAttribute(
			'aria-current',
			'page'
		);
		await expect(page.getByRole('heading', { level: 3, name: 'Staging' })).toBeVisible();
		await expect(page.getByRole('heading', { level: 3, name: 'Production' })).toBeVisible();
		await expect(page.getByText('Deploy to staging first, then promote it here')).toBeVisible();
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
		await drawer.getByLabel('Name').fill('e2e registry');
		await drawer.getByLabel('Url').fill('https://registry.e2e.test');
		await shot(page, 'registry-drawer');
		await drawer.getByRole('button', { name: 'Create registry' }).click();
		await expect(drawer).toHaveCount(0);

		const row = page.getByRole('row').filter({ hasText: 'e2e registry' });
		await expect(row).toBeVisible();
		await row.click();
		const edit = page.getByRole('dialog', { name: 'e2e registry' });
		await expect(edit.getByLabel('Url')).toHaveValue('https://registry.e2e.test');
		await edit.getByRole('button', { name: 'Delete registry' }).click();
		const dialog = page.getByRole('alertdialog');
		await dialog.getByLabel('Type e2e registry to confirm').fill('e2e registry');
		await dialog.getByRole('button', { name: 'Delete registry' }).click();
		await expect(row).toHaveCount(0);
	});

	test('jobs: rows open their details in a drawer', async ({ page }) => {
		await signIn(page, '/jobs/');
		await expect(page.getByRole('heading', { level: 1, name: 'Jobs' })).toBeVisible();
		const rows = page.getByRole('row').filter({ has: page.getByRole('button') });
		if ((await rows.count()) === 0) {
			await expect(page.getByText('The queue is empty')).toBeVisible();
			return;
		}
		await rows.first().getByRole('button').first().click();
		const drawer = page.getByRole('dialog');
		await expect(drawer.getByRole('heading', { name: 'Payload' })).toBeVisible();
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
		await shot(page, 'deployments-menu');
		await page.keyboard.press('Escape');

		await page.getByRole('searchbox', { name: 'Branch, commit or #number' }).fill('no-such-branch');
		await expect(page.getByText('No deployment matches')).toBeVisible();
		await page.getByRole('searchbox', { name: 'Branch, commit or #number' }).fill('');
		await expect(row).toBeVisible();
		await shot(page, 'deployments-tab');
	});

	test('every page renders without errors; unknown ids show not found', async ({ page }) => {
		const errors = watchErrors(page);
		await signIn(page);
		const { app } = await fixtures(page);

		for (const path of [
			'/jobs/',
			'/apps/new/',
			'/registries/new/',
			`/apps/${app}/`,
			`/apps/${app}/edit/`,
			`/apps/${app}/deployments/new/`,
			`/apps/${app}/deployments/`
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
