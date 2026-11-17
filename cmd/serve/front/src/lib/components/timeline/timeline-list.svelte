<!--
	A list with no groups (DESIGN.md §5 Lists): applications, targets, registries, services,
	volumes. Rows sit 2px apart, no lines, no column headings; each row says what it is in its own
	words. The list bleeds 8px (`-mx-2`), so a row's tile lines up with the page's left edge and the
	hover wash spills just past it.
-->
<script lang="ts">
	import type { Snippet } from 'svelte';

	type Props = {
		/** The list's accessible name, when no heading names it. */
		label?: string;
		labelledby?: string;
		children: Snippet;
	};

	let { label, labelledby, children }: Props = $props();
</script>

<ul aria-label={label} aria-labelledby={labelledby} class="-mx-2 flex flex-col gap-0.5">
	{@render children()}
</ul>
