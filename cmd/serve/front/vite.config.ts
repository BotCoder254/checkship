import tailwindcss from '@tailwindcss/vite';
import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import { cssModules } from 'svelte-preprocess-cssmodules';
import { sveltekit } from '@sveltejs/kit/vite';

import { defineConfig } from 'vitest/config';

export default defineConfig({
	plugins: [
		tailwindcss(),
		sveltekit({
			// TypeScript first, then CSS modules (`<style module>`), which must run last.
			preprocess: [vitePreprocess(), cssModules()],
			alias: { $components: 'src/components', $assets: 'src/assets' },
			adapter: adapter({
				fallback:
					'fallback.html' /* Enable true SPA mode since some pages could not be pregenerated (ex: apps pages) */
			})
		})
	],
	server: {
		proxy: {
			// During development, proxy api requests to the go server to make things transparent
			'/api/v1': 'http://127.0.0.1:8080'
		}
	},
	test: {
		include: ['src/**/*.{test,spec}.{js,ts}']
	}
});
