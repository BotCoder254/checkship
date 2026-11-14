import { defineConfig, devices } from '@playwright/test';
import { mkdtempSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

/**
 * End-to-end tests run against the real app: the Go binary (`go run . serve` at the repository
 * root) serving the embedded dashboard from `build/`, so run `npm run build` first
 * (`npm run test:e2e` does both). Every run gets a fresh data directory, so a fresh database with
 * the admin account below.
 */
const port = Number(process.env.E2E_PORT ?? 8099);

// Workers inherit the main process environment, so they all see the same directory.
process.env.E2E_DATA_PATH ??= mkdtempSync(join(tmpdir(), 'checkship-e2e-'));

export default defineConfig({
	testDir: 'e2e',
	fullyParallel: false,
	workers: 1,
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 1 : 0,
	reporter: [['list'], ['html', { open: 'never' }]],
	use: {
		baseURL: `http://localhost:${port}`,
		trace: 'retain-on-failure',
		screenshot: 'only-on-failure'
	},
	projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
	webServer: {
		command: 'go run . serve',
		cwd: '../../..',
		url: `http://localhost:${port}/api/v1/healthcheck`,
		reuseExistingServer: false,
		timeout: 300_000,
		stdout: 'ignore',
		stderr: 'pipe',
		env: {
			HTTP_PORT: String(port),
			// Plain http on localhost: the session cookie must not be Secure.
			HTTP_SECURE: 'false',
			DATA_PATH: process.env.E2E_DATA_PATH,
			ADMIN_EMAIL: 'admin@example.com',
			ADMIN_PASSWORD: 'admin',
			LOG_LEVEL: 'error'
		}
	}
});
