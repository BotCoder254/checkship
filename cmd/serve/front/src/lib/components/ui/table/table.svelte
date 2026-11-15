<script lang="ts">
	import { cn, type WithElementRef } from '#lib/utils.js';
	import type { HTMLTableAttributes } from 'svelte/elements';

	let {
		ref = $bindable(null),
		class: className,
		containerClass,
		children,
		...restProps
	}: WithElementRef<HTMLTableAttributes> & { containerClass?: string } = $props();
</script>

<!-- DESIGN.md §5 Tables: the container is a size container, so columns drop by priority
	(see #lib/table-columns.svelte.ts) instead of the table scrolling sideways. -->
<div
	data-slot="table-container"
	class={cn(
		'@container relative w-full rounded-lg border border-line-strong px-1.5 py-1',
		containerClass
	)}
>
	<table
		bind:this={ref}
		data-slot="table"
		class={cn(
			'w-full table-fixed caption-bottom border-separate border-spacing-0 text-[13px] text-ink',
			className
		)}
		{...restProps}
	>
		{@render children?.()}
	</table>
</div>
