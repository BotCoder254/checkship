import js from '@eslint/js';
import prettier from 'eslint-config-prettier';
import svelte from 'eslint-plugin-svelte';
import { defineConfig, globalIgnores } from 'eslint/config';
import globals from 'globals';
import ts from 'typescript-eslint';

export default defineConfig(
	globalIgnores(['build/', '.svelte-kit/', 'node_modules/', 'test-results/', 'playwright-report/']),
	js.configs.recommended,
	...ts.configs.recommended,
	...svelte.configs.recommended,
	prettier,
	...svelte.configs.prettier,
	{
		languageOptions: { globals: { ...globals.browser, ...globals.node } },
		rules: {
			// typescript-eslint recommends turning this off for TypeScript projects: the compiler already checks it.
			'no-undef': 'off',
			// checkship is served at the root (no base path) and every route comes from #lib/path.ts,
			// so wrapping links in resolve() adds nothing.
			'svelte/no-navigation-without-resolve': 'off'
		}
	},
	{
		// Generated shadcn-svelte components take an href from the caller, which resolves it.
		files: ['src/lib/components/ui/**'],
		rules: { 'svelte/no-navigation-without-resolve': 'off' }
	},
	{
		files: ['**/*.svelte', '**/*.svelte.ts', '**/*.svelte.js'],
		languageOptions: {
			parserOptions: {
				projectService: true,
				extraFileExtensions: ['.svelte'],
				parser: ts.parser
			}
		}
	}
);
