<script lang="ts">
	import { untrack } from 'svelte';
	import { preferences } from '#lib/preferences.svelte.ts';

	let { data, children } = $props();

	// Opening an application (or one of its deployments) puts it at the top of Recent in the sidebar.
	// Only the id is tracked: rememberApp reads the list it writes, which would re-run this forever.
	$effect(() => {
		const id = data.app.id;
		untrack(() => preferences.rememberApp(id));
	});
</script>

{@render children()}
