// svelte-preprocess-cssmodules ships dist/index.d.ts but its package.json "exports" map has no
// "types" condition, so TypeScript can't find it. Remove this file once `<style module>` is gone.
declare module 'svelte-preprocess-cssmodules' {
	import type { PreprocessorGroup } from 'svelte/compiler';

	export function cssModules(options?: Record<string, unknown>): PreprocessorGroup;
}
