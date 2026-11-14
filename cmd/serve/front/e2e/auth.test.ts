import { expect, test, type Page } from '@playwright/test';

/**
 * Sign-in screen (DESIGN.md §5 Auth layout, §11.5 patterns) against the real Go API.
 * Each test takes a screenshot into its output folder (test-results/) for review.
 */

const ADMIN = { email: 'admin@example.com', password: 'admin' };

// Palette tokens (DESIGN.md §3) as rgb, the way the browser reports them.
const DARK = {
	canvas: [11, 11, 13],
	accent: [127, 149, 255],
	faint: 'rgb(126, 131, 143)',
	onAccent: 'rgb(13, 16, 32)'
};
const LIGHT = { canvas: [238, 240, 243], accent: [47, 91, 234], faint: 'rgb(138, 145, 156)' };

const key = (rgb: number[]) => rgb.join(',');

/** The distinct colours painted on the dither canvas (sampled), once it has been drawn. */
async function ditherColours(page: Page): Promise<string[]> {
	return page.locator('[data-slot="auth-art"] canvas').evaluate((canvas: HTMLCanvasElement) => {
		if (canvas.width === 0 || canvas.height === 0) return [];
		const { data } = canvas.getContext('2d')!.getImageData(0, 0, canvas.width, canvas.height);
		const seen = new Set<string>();
		for (let i = 0; i < data.length; i += 4 * 101) {
			if (data[i + 3] === 0) continue;
			seen.add(`${data[i]},${data[i + 1]},${data[i + 2]}`);
		}
		return [...seen].sort();
	});
}

async function box(page: Page, selector: string) {
	const rect = await page.locator(selector).first().boundingBox();
	expect(rect, `${selector} has a box`).not.toBeNull();
	return rect!;
}

async function noHorizontalScroll(page: Page) {
	const overflow = await page.evaluate(
		() => document.documentElement.scrollWidth - document.documentElement.clientWidth
	);
	expect(overflow).toBeLessThanOrEqual(0);
}

async function expectFormCentred(page: Page, area: { x: number; width: number; height: number }) {
	const form = await box(page, 'form');
	expect(Math.abs(form.x + form.width / 2 - (area.x + area.width / 2))).toBeLessThanOrEqual(1);
	expect(Math.abs(form.y + form.height / 2 - area.height / 2)).toBeLessThanOrEqual(1);
}

/** Fails the test on uncaught page errors. Failed API calls (401, 400) are expected and ignored. */
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

async function shot(page: Page, name: string) {
	await page.screenshot({ path: test.info().outputPath(`${name}.png`), fullPage: true });
}

test.describe('layout', () => {
	test('desktop 1440: art panel left, form centred right, no divider, dark by default', async ({
		page
	}) => {
		const errors = watchErrors(page);
		await page.setViewportSize({ width: 1440, height: 900 });
		await page.goto('/signin/');

		await expect(page).toHaveTitle('checkship - Sign in');
		await expect(page.locator('html')).toHaveClass(/dark/);
		await expect(page.getByRole('heading', { level: 1, name: 'Sign in' })).toBeVisible();
		await expect(page.getByText('Painless self-hosted deployment platform.')).toBeVisible();

		// Two full-height columns, half and half.
		const panel = await box(page, '[data-slot="auth-art"]');
		const main = await box(page, 'main');
		expect(panel).toEqual({ x: 0, y: 0, width: 720, height: 900 });
		expect(main).toEqual({ x: 720, y: 0, width: 720, height: 900 });

		// Centred both ways in the right column, 360px wide.
		expect((await box(page, 'form')).width).toBe(360);
		await expectFormCentred(page, main);

		// No divider: no border or shadow on either side of the boundary.
		for (const selector of ['[data-slot="auth-art"]', 'main']) {
			const edges = await page.locator(selector).evaluate((el) => {
				const s = getComputedStyle(el);
				return [s.borderLeftWidth, s.borderRightWidth, s.boxShadow];
			});
			expect(edges).toEqual(['0px', '0px', 'none']);
		}

		// The art is dithered into exactly the two dark tokens.
		await expect
			.poll(() => ditherColours(page))
			.toEqual([key(DARK.canvas), key(DARK.accent)].sort());
		await expect(page.locator('[data-slot="auth-art"] > div')).toHaveAttribute(
			'aria-hidden',
			'true'
		);

		await noHorizontalScroll(page);
		await shot(page, 'desktop-1440-dark');
		expect(errors).toEqual([]);
	});

	test('dark is the default even when the OS is light, until someone picks a theme', async ({
		browser
	}) => {
		const context = await browser.newContext({ colorScheme: 'light' });
		const page = await context.newPage();
		await page.goto('/signin/');

		await expect(page.locator('html')).toHaveClass(/dark/);
		expect(await page.evaluate(() => localStorage.getItem('mode-watcher-mode'))).toBe('dark');
		await context.close();
	});

	test('desktop 1440, light mode: the art follows the light tokens', async ({ page }) => {
		await page.addInitScript(() => localStorage.setItem('mode-watcher-mode', 'light'));
		await page.setViewportSize({ width: 1440, height: 900 });
		await page.goto('/signin/');

		await expect(page.locator('html')).not.toHaveClass(/dark/);
		await expect
			.poll(() => ditherColours(page))
			.toEqual([key(LIGHT.canvas), key(LIGHT.accent)].sort());
		await expect(page.getByLabel('Password')).toHaveCSS('border-top-color', LIGHT.faint);
		await shot(page, 'desktop-1440-light');
	});

	test('wide 1920: the panel keeps half the width and the art is not cut', async ({ page }) => {
		await page.setViewportSize({ width: 1920, height: 1080 });
		await page.goto('/signin/');

		expect(await box(page, '[data-slot="auth-art"]')).toEqual({
			x: 0,
			y: 0,
			width: 960,
			height: 1080
		});
		await expectFormCentred(page, await box(page, 'main'));
		await expect
			.poll(() => ditherColours(page))
			.toEqual([key(DARK.canvas), key(DARK.accent)].sort());
		await shot(page, 'wide-1920-dark');
	});

	test('tablet 834: no panel, form centred', async ({ page }) => {
		await page.setViewportSize({ width: 834, height: 1112 });
		await page.goto('/signin/');

		await expect(page.locator('[data-slot="auth-art"]')).toBeHidden();
		await expectFormCentred(page, { x: 0, width: 834, height: 1112 });
		await expect(page.getByLabel('Email')).toHaveCSS('height', '36px');
		await noHorizontalScroll(page);
		await shot(page, 'tablet-834');
	});

	test('phone 390: no panel, 44px controls with 16px text, nothing overflows', async ({ page }) => {
		await page.setViewportSize({ width: 390, height: 844 });
		await page.goto('/signin/');

		await expect(page.locator('[data-slot="auth-art"]')).toBeHidden();
		for (const control of [page.getByLabel('Email'), page.getByLabel('Password')]) {
			await expect(control).toHaveCSS('height', '44px');
			await expect(control).toHaveCSS('font-size', '16px');
		}
		await expect(page.getByRole('button', { name: 'Sign in' })).toHaveCSS('height', '44px');
		expect((await box(page, 'form')).width).toBe(390 - 48);
		await noHorizontalScroll(page);
		await shot(page, 'phone-390');
	});
});

test.describe('controls', () => {
	test.use({ viewport: { width: 1440, height: 900 } });

	test('inputs are visible: faint border, sunken fill', async ({ page }) => {
		await page.goto('/signin/');
		const password = page.getByLabel('Password');

		await expect(password).toHaveCSS('border-top-color', DARK.faint);
		await expect(password).toHaveCSS('background-color', 'rgb(15, 16, 18)');
	});

	test('focus is drawn inside the control, never as an outer ring', async ({ page }) => {
		await page.goto('/signin/');
		await expect(page.getByLabel('Email')).toBeFocused();

		await page.keyboard.press('Tab');
		const password = page.getByLabel('Password');
		await expect(password).toBeFocused();
		await expect(password).toHaveCSS(
			'border-top-color',
			`rgb(${key(DARK.accent).replaceAll(',', ', ')})`
		);
		await expect(password).toHaveCSS('outline-style', 'none');
		expect(await password.evaluate((el) => getComputedStyle(el).boxShadow)).toContain('inset');

		await page.keyboard.press('Tab');
		const button = page.getByRole('button', { name: 'Sign in' });
		await expect(button).toBeFocused();
		await expect(button).toHaveCSS('outline-style', 'none');
		// The button animates its ring (150ms), so wait for the settled value: 2px, on-accent, inset.
		await expect
			.poll(() => button.evaluate((el) => getComputedStyle(el).boxShadow))
			.toContain(`${DARK.onAccent} 0px 0px 0px 2px inset`);
		await shot(page, 'focus-button');
	});
});

test.describe('sign in', () => {
	test.use({ viewport: { width: 1440, height: 900 } });

	test('wrong password: one alert, both fields invalid', async ({ page }) => {
		const errors = watchErrors(page);
		await page.goto('/signin/');
		await page.getByLabel('Email').fill(ADMIN.email);
		await page.getByLabel('Password').fill('wrong-password');
		await page.getByRole('button', { name: 'Sign in' }).click();

		const alert = page.locator('[data-slot="alert"]');
		await expect(alert).toHaveText('Invalid email or password');
		await expect(alert).toHaveCSS('background-color', 'rgb(58, 29, 27)');
		await expect(page.getByLabel('Email')).toHaveAttribute('aria-invalid', 'true');
		await expect(page.getByLabel('Password')).toHaveAttribute('aria-invalid', 'true');
		await expect(page.locator('[data-slot="field-error"]')).toHaveCount(0);
		await expect(page).toHaveURL(/\/signin\/$/);
		await shot(page, 'error-credentials');
		expect(errors).toEqual([]);
	});

	test('field error: shown under the field, cleared when the field is edited', async ({ page }) => {
		await page.goto('/signin/');
		await page.getByLabel('Password').fill('anything');
		await page.getByRole('button', { name: 'Sign in' }).click();

		const email = page.getByLabel('Email');
		const message = page.locator('#email-error');
		await expect(message).toHaveText('Invalid email');
		await expect(email).toHaveAttribute('aria-invalid', 'true');
		await expect(email).toHaveAttribute('aria-describedby', 'email-error');
		await shot(page, 'error-field');

		await email.fill('a');
		await expect(message).toHaveCount(0);
		await expect(email).not.toHaveAttribute('aria-invalid', 'true');
	});

	test('loading: spinner next to the kept label, focus kept, one request only', async ({
		page
	}) => {
		let requests = 0;
		await page.route('**/api/v1/sessions', async (route) => {
			requests++;
			await new Promise((resolve) => setTimeout(resolve, 1500));
			await route.continue();
		});

		await page.goto('/signin/');
		await page.getByLabel('Email').fill(ADMIN.email);
		await page.getByLabel('Password').fill(ADMIN.password);
		const button = page.getByRole('button', { name: 'Sign in' });
		const width = (await button.boundingBox())!.width;
		await button.focus();
		await page.keyboard.press('Enter');

		await expect(button).toHaveAttribute('aria-busy', 'true');
		await expect(button.locator('svg.animate-spin')).toBeVisible();
		await expect(button).toHaveText('Sign in');
		await expect(button).toBeFocused();
		await expect(button).toBeEnabled();
		expect((await button.boundingBox())!.width).toBe(width);
		await expect(page.getByLabel('Email')).toHaveAttribute('readonly', '');
		await shot(page, 'loading');

		// A second click and another Enter while pending do nothing.
		await button.click();
		await page.getByLabel('Password').press('Enter');

		await expect(page).toHaveURL(/\/$/);
		expect(requests).toBe(1);
	});

	test('success: lands on the overview', async ({ page }) => {
		const errors = watchErrors(page);
		await page.goto('/signin/');
		await page.getByLabel('Email').fill(ADMIN.email);
		await page.getByLabel('Password').fill(ADMIN.password);
		await page.getByRole('button', { name: 'Sign in' }).click();

		await expect(page).toHaveURL(/\/$/);
		await expect(page.getByRole('button', { name: `Account: ${ADMIN.email}` })).toBeVisible();
		await shot(page, 'after-sign-in');
		expect(errors).toEqual([]);
	});

	test('a protected page sends you to sign in, then back to it', async ({ page }) => {
		await page.goto('/targets/');
		await expect(page).toHaveURL(/\/signin\/\?redirectTo=%2Ftargets%2F$/);

		await page.getByLabel('Email').fill(ADMIN.email);
		await page.getByLabel('Password').fill(ADMIN.password);
		await page.getByRole('button', { name: 'Sign in' }).click();
		await expect(page).toHaveURL(/\/targets\/$/);
	});

	test('an unknown redirectTo falls back to the overview', async ({ page }) => {
		await page.goto('/signin/?redirectTo=%2Fdoes-not-exist%2F');
		await page.getByLabel('Email').fill(ADMIN.email);
		await page.getByLabel('Password').fill(ADMIN.password);
		await page.getByRole('button', { name: 'Sign in' }).click();
		await expect(page).toHaveURL(/\/$/);
	});
});
