<!-- One group of a row menu's items; groups are split by the menu's 6px gap, never a line. -->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';

	let { children }: { children: Snippet } = $props();
</script>

<DropdownMenu.Group class="flex flex-col">
	{@render children()}
</DropdownMenu.Group>
