<!--
	A section's label inside a page or drawer (DESIGN.md §5 Section headings): small, semibold,
	`muted`, sentence case, a little letter-spacing, like the sidebar's "Workspace" label. It names
	the section without competing with what is in it; space, not a box, sets the section apart.
	An `h2` by default; `level` for an `h3` inside a section.
-->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import { cn } from '#lib/utils.js';

	type Props = {
		id?: string;
		level?: 2 | 3;
		class?: string;
		children: Snippet;
		/** Beside the heading on the right: a count, a link, a small action. */
		aside?: Snippet;
	};

	let { id, level = 2, class: className, children, aside }: Props = $props();
</script>

<div class={cn('flex min-h-6 flex-wrap items-center justify-between gap-x-3 gap-y-1', className)}>
	<svelte:element
		this={`h${level}`}
		{id}
		class="flex items-center gap-2 text-xs font-semibold tracking-wide text-muted-foreground"
	>
		{@render children()}
	</svelte:element>
	{#if aside}
		<div class="flex flex-wrap items-center gap-2 text-label text-muted-foreground">
			{@render aside()}
		</div>
	{/if}
</div>
