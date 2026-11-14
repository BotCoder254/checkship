<script lang="ts">
	import AppShell from '#lib/components/shell/app-shell.svelte';

	let { data, children } = $props();
</script>

<AppShell user={data.user} version={data.health.version}>
	{@render children()}
</AppShell>
