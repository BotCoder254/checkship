import { expect, test, type Page } from '@playwright/test';

/**
 * Signed-in shell and the rebuilt pages (DESIGN.md §5) against the real Go API. Screenshots go to
 * each test's output folder (test-results/) for review.
 */

const ADMIN = { email: 'admin@example.com', password: 'admin' };

async function signIn(page: Page, path = '/') {
	await page.goto(`/signin/?redirectTo=${encodeURIComponent(path)}`);
	await page.getByLabel('Email').fill(ADMIN.email);
	await page.getByLabel('Password').fill(ADMIN.password);
	await page.getByRole('button', { name: 'Sign in' }).click();
	await expect(page).toHaveURL(new RegExp(`${path.replaceAll('/', '\\/')}$`));
}

async function shot(page: Page, name: string) {
	await page.screenshot({ path: test.info().outputPath(`${name}.png`), fullPage: true });
}

/** No line on a box: zero-width borders, or borders drawn in a transparent colour. */
async function expectNoLines(page: Page, selector: string) {
	const edges = await page.locator(selector).evaluateAll((elements) =>
		elements.map((el) => {
			const s = getComputedStyle(el);
			return ['Top', 'Right', 'Bottom', 'Left'].every((side) => {
				const width = s.getPropertyValue(`border-${side.toLowerCase()}-width`);
				const color = s.getPropertyValue(`border-${side.toLowerCase()}-color`);
				return width === '0px' || color === 'rgba(0, 0, 0, 0)';
			});
		})
	);
	expect(edges.length).toBeGreaterThan(0);
	expect(edges.every(Boolean)).toBe(true);
}

test.describe('desktop shell', () => {
	test.use({ viewport: { width: 1440, height: 900 } });

	test('title bar, sidebar and workspace, with no lines between them', async ({ page }) => {
		await signIn(page);

		const header = page.locator('header').first();
		const headerBox = (await header.boundingBox())!;
		expect(headerBox.height).toBe(56);

		// The logo sits on the sidebar toggle's centre line, both clear of the window's top edge.
		const toggle = (await page.getByRole('button', { name: 'Collapse sidebar' }).boundingBox())!;
		const logo = (await page.getByRole('link', { name: 'checkship' }).first().boundingBox())!;
		expect(Math.abs(logo.y + logo.height / 2 - (toggle.y + toggle.height / 2))).toBeLessThanOrEqual(
			1
		);
		expect(toggle.y).toBeGreaterThanOrEqual(10);

		const avatar = page.getByRole('button', { name: `Account: ${ADMIN.email}` });
		await expect(avatar).toBeVisible();
		await expect(avatar).toHaveCSS('border-radius', /^(18px|50%|9999px|3.35544e\+07px)$/);
		const avatarBox = (await avatar.boundingBox())!;
		expect(avatarBox.width).toBe(36);
		expect(avatarBox.x + avatarBox.width).toBeGreaterThan(1400);
		// The avatar keeps at least 8px from the workspace panel below it.
		const mainTop = (await page.locator('main').boundingBox())!.y;
		expect(mainTop - (avatarBox.y + avatarBox.height)).toBeGreaterThanOrEqual(8);

		// The language menu sits just left of the avatar.
		const language = (await page.getByRole('button', { name: /^Language:/ }).boundingBox())!;
		expect(language.x + language.width).toBeLessThanOrEqual(avatarBox.x);
		expect(avatarBox.x - (language.x + language.width)).toBeLessThanOrEqual(12);

		const aside = page.locator('aside');
		expect((await aside.boundingBox())!.width).toBe(240);

		// The nav starts 24px below the title bar, under a Workspace label.
		const workspace = page.getByText('Workspace', { exact: true });
		expect((await workspace.boundingBox())!.y).toBeGreaterThanOrEqual(56 + 24);

		// Documentation and the theme switch sit at the bottom of the sidebar.
		const theme = page.getByRole('radiogroup', { name: 'Theme' });
		expect((await theme.boundingBox())!.y).toBeGreaterThan(600);
		await expect(page.getByRole('link', { name: 'Documentation' })).toBeVisible();
		await expect(page.locator('aside').getByText(/^checkship \d/)).toBeVisible();

		const main = page.locator('main');
		await expect(main).toHaveCSS('border-top-left-radius', '6px');
		await expectNoLines(page, 'header, aside, main, nav');

		await expect(page.locator('aside').getByRole('link', { name: 'Overview' })).toHaveAttribute(
			'aria-current',
			'page'
		);
		await shot(page, 'overview-shell');
	});

	test('the sidebar collapses to icons and remembers it', async ({ page }) => {
		await signIn(page);
		await page.getByRole('button', { name: 'Collapse sidebar' }).click();
		expect((await page.locator('aside').boundingBox())!.width).toBe(56);

		await page.reload();
		expect((await page.locator('aside').boundingBox())!.width).toBe(56);
		await page.getByRole('button', { name: 'Expand sidebar' }).click();
		expect((await page.locator('aside').boundingBox())!.width).toBe(240);
	});

	test('the theme switch changes and keeps the theme', async ({ page }) => {
		await signIn(page);
		await expect(page.locator('html')).toHaveClass(/dark/);

		await page.getByRole('radio', { name: 'Light' }).click();
		await expect(page.locator('html')).not.toHaveClass(/dark/);
		await expect(page.getByRole('radio', { name: 'Light' })).toHaveAttribute(
			'aria-checked',
			'true'
		);
		await shot(page, 'applications-light');

		await page.reload();
		await expect(page.locator('html')).not.toHaveClass(/dark/);
	});

	test('the language menu lists the languages and switches to one', async ({ page }) => {
		await signIn(page);
		await page.getByRole('button', { name: 'Language: English' }).click();

		await expect(page.getByRole('menuitem', { name: /English/ })).toHaveAttribute(
			'aria-current',
			'true'
		);
		await shot(page, 'language-menu');

		await page.getByRole('menuitem', { name: /Français/ }).click();
		await expect(page.getByRole('button', { name: 'Langue : Français' })).toBeVisible();

		// Back to English for the other tests
		await page.getByRole('button', { name: 'Langue : Français' }).click();
		await page.getByRole('menuitem', { name: /English/ }).click();
		await expect(page.getByRole('button', { name: 'Language: English' })).toBeVisible();
	});

	test('the account menu opens from the avatar and signs out', async ({ page }) => {
		await signIn(page);
		await page.getByRole('button', { name: `Account: ${ADMIN.email}` }).click();

		const menu = page.getByRole('menu');
		await expect(menu).toContainText(ADMIN.email);
		await expect(page.getByRole('menuitem', { name: 'Profile' })).toBeVisible();
		await expect(page.getByRole('menuitem', { name: 'API key for CI' })).toBeVisible();
		await shot(page, 'account-menu');

		await page.getByRole('menuitem', { name: 'Sign out' }).click();
		await expect(page).toHaveURL(/\/signin\//);
	});
});

test.describe('pages', () => {
	test.use({ viewport: { width: 1440, height: 900 } });

	test('targets: create one, see it as a row, delete asks for its name', async ({ page }) => {
		await signIn(page, '/targets/');
		await page.getByRole('link', { name: 'New target' }).first().click();
		await expect(page).toHaveURL(/\/targets\/new\/$/);

		await page.getByLabel('Name').fill('e2e-local');
		await page.getByRole('switch', { name: 'Expose services automatically' }).click();
		await expect(page.getByLabel('Url')).toHaveCount(0);
		await shot(page, 'target-new');

		const create = page.getByRole('button', { name: 'Create target' });
		await expect(create.locator('svg')).toHaveCount(1);
		await create.click();
		// Creating opens the target's overview.
		await expect(page).toHaveURL(/\/targets\/[^/]+\/$/);
		await expect(page.getByRole('heading', { level: 1, name: 'e2e-local' })).toBeVisible();
		await page.goto('/targets/');

		const row = page.getByRole('main').getByRole('listitem').filter({ hasText: 'e2e-local' });
		await expect(row).toBeVisible();
		await expect(row).toContainText('This machine · Docker engine');
		// Figures and the spread over targets sit above the list (DESIGN.md §11.8).
		await expect(page.getByRole('region', { name: 'Targets at a glance' })).toBeVisible();
		await expectNoLines(page, 'main li');
		// Every list is a list now (DESIGN.md §5 Lists): no tables, no Columns menu.
		await expect(page.getByRole('main').getByRole('table')).toHaveCount(0);
		await expect(page.getByRole('button', { name: 'Columns' })).toHaveCount(0);
		await shot(page, 'targets');

		// Delete is disabled while a target is configuring; wait until configuration has finished
		// (it fails here: this machine has no Docker), then open its settings.
		const targetId = (await row.getByRole('link').getAttribute('href'))!.split('/')[2];
		await expect
			.poll(
				async () =>
					(await (await page.request.get(`/api/v1/targets/${targetId}`)).json()).state.status,
				{ timeout: 20_000 }
			)
			.not.toBe(0);
		await page.goto(`/targets/${targetId}/edit/`);
		await page.getByRole('button', { name: 'Delete target' }).click();
		const dialog = page.getByRole('alertdialog');
		const confirm = dialog.getByRole('button', { name: 'Delete target' });
		await expect(confirm).toBeDisabled();
		await dialog.getByLabel('Target name').fill('e2e-local');
		await expect(confirm).toBeEnabled();
		await shot(page, 'target-delete-dialog');
		await dialog.getByRole('button', { name: 'Cancel' }).click();
		await expect(dialog).toHaveCount(0);
	});

	test('recent applications: an opened application shows in the sidebar', async ({ page }) => {
		await signIn(page, '/targets/');
		const targets = await (await page.request.get('/api/v1/targets')).json();
		const target = targets[0]?.id;
		test.skip(!target, 'needs the target created by the previous test');

		const created = await page.request.post('/api/v1/apps', {
			data: { name: 'e2e-app', production: { target }, staging: { target } }
		});
		expect(created.ok()).toBe(true);
		const app = await created.json();

		await page.goto('/apps/');
		const row = page.getByRole('main').getByRole('listitem').filter({ hasText: 'e2e-app' });
		await expect(row).toContainText('No deployment yet');
		await expect(page.getByText('Recent', { exact: true })).toHaveCount(0);

		await row.click();
		await expect(page).toHaveURL(new RegExp(`/apps/${app.id}/$`));
		await expect(page.getByText('Not found')).toHaveCount(0);
		await expect(page.getByText('Recent', { exact: true })).toBeVisible();
		await expect(page.locator('aside').getByRole('link', { name: 'e2e-app' })).toBeVisible();
		await page.goto('/apps/');
		await shot(page, 'applications-with-recent');
	});

	test('profile: sections save on their own, the key is masked', async ({ page }) => {
		await signIn(page, '/profile/');
		await expect(page.getByRole('heading', { level: 1, name: 'Profile' })).toBeVisible();

		for (const name of [
			'Sign out',
			'Save email',
			'Change password',
			'Reveal',
			'Copy',
			'Regenerate'
		]) {
			await expect(
				page.getByRole('button', { name, exact: true }).first().locator('svg')
			).toHaveCount(1);
		}

		// Nothing to save until the email or the password is typed.
		await expect(page.getByRole('button', { name: 'Save email' })).toBeDisabled();
		await expect(page.getByRole('button', { name: 'Change password' })).toBeDisabled();

		// The password is typed twice and checked here, without calling the API.
		await page.getByLabel('New password').fill('one');
		await page.getByLabel('Repeat it').fill('two');
		await page.getByRole('button', { name: 'Change password' }).click();
		await expect(page.getByText('The two passwords don’t match.')).toBeVisible();

		// The key is masked until revealed.
		const key = (await (await page.request.get('/api/v1/profile')).json()).api_key as string;
		await expect(page.getByText(key, { exact: true })).toHaveCount(0);
		await page.getByRole('button', { name: 'Reveal' }).click();
		await expect(page.getByText(key, { exact: true })).toBeVisible();
		await page.getByRole('button', { name: 'Hide' }).click();

		await page.getByRole('button', { name: 'Regenerate', exact: true }).click();
		await expect(page.getByRole('alertdialog')).toContainText('Regenerate the API key?');
		await shot(page, 'profile-regenerate');
		await page.getByRole('alertdialog').getByRole('button', { name: 'Cancel' }).click();
		await shot(page, 'profile');
	});

	test('pages fill the workspace: no max width, not centred', async ({ page }) => {
		await page.setViewportSize({ width: 1920, height: 1000 });
		await signIn(page, '/jobs/');
		const main = (await page.locator('main').boundingBox())!;
		const title = (await page.getByRole('heading', { level: 1 }).boundingBox())!;
		const pause = (await page.getByRole('button', { name: 'Pause updates' }).boundingBox())!;
		// 40px padding on every side from 1024px: the title starts at the left edge, the header
		// button ends at the right.
		expect(Math.round(title.x - main.x)).toBe(40);
		expect(Math.round(main.x + main.width - (pause.x + pause.width))).toBe(40);
		// And 40px above the header buttons.
		expect(Math.round(pause.y - main.y)).toBe(40);
	});
});

test.describe('smaller screens', () => {
	test('tablet 834: icon sidebar, the menu button opens the full menu', async ({ page }) => {
		await page.setViewportSize({ width: 834, height: 1112 });
		await signIn(page);
		expect((await page.locator('aside').boundingBox())!.width).toBe(56);

		await page.getByRole('button', { name: 'Open menu' }).click();
		const sheet = page.getByRole('dialog');
		await expect(sheet.getByText('Workspace')).toBeVisible();
		await expect(sheet.getByRole('radiogroup', { name: 'Theme' })).toBeVisible();
		await shot(page, 'tablet-menu');
	});

	test('phone 390: no sidebar, 44px menu button, 40px avatar, no sideways scroll', async ({
		page
	}) => {
		await page.setViewportSize({ width: 390, height: 844 });
		await signIn(page);
		await expect(page.locator('aside')).toBeHidden();

		const menu = page.getByRole('button', { name: 'Open menu' });
		expect((await menu.boundingBox())!.height).toBe(44);
		expect(
			(await page.getByRole('button', { name: `Account: ${ADMIN.email}` }).boundingBox())!.width
		).toBe(40);
		const overflow = await page.evaluate(
			() => document.documentElement.scrollWidth - document.documentElement.clientWidth
		);
		expect(overflow).toBeLessThanOrEqual(0);
		await shot(page, 'phone');

		await menu.click();
		await page.getByRole('dialog').getByRole('link', { name: 'Targets' }).click();
		await expect(page).toHaveURL(/\/targets\/$/);
		await expect(page.getByRole('dialog')).toHaveCount(0);
	});
});
