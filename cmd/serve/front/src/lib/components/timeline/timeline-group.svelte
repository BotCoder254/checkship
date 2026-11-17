<!--
	One group of a list (DESIGN.md §5 Lists): a day ("Today · 4") or a state ("Running · 1")
	heading, then its rows. The heading sticks to the top of the scroll area while its rows pass
	under it, on a solid fill (no blur): `muted`, sentence case. Rows sit 2px apart, no lines. The
	group bleeds 8px (`-mx-2`), so the heading text and a row's tile line up with the page's left
	edge and the hover wash spills just past it.
-->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '#lib/utils.js';

	type Props = {
		label: string;
		count?: number;
		/** Stick the heading while scrolling (off for short lists in a tile). */
		sticky?: boolean;
		children: Snippet;
	};

	let { label, count, sticky = true, children }: Props = $props();

	const headingId = $props.id();
</script>

<section aria-labelledby={headingId} class="-mx-2 flex flex-col">
	<h3
		id={headingId}
		class={cn(
			'z-20 flex items-center gap-1.5 px-2 pt-1 pb-1.5 text-xs font-semibold tracking-wide text-muted-foreground',
			sticky && 'sticky top-0 bg-surface'
		)}
	>
		{label}
		{#if count !== undefined}
			<span class="font-normal tabular-nums">· {count}</span>
		{/if}
	</h3>
	<ul class="flex flex-col gap-0.5">
		{@render children()}
	</ul>
</section>
