<script lang="ts">
	import { cn, type WithElementRef } from '#lib/utils.js';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLTableSectionElement>> = $props();
</script>

<tbody
	bind:this={ref}
	data-slot="table-body"
	class={cn(
		// The hovered row and the row whose ⋯ menu is open get a rounded fill on their cells.
		'[&>tr:has([aria-expanded=true])>*]:bg-raised [&>tr:hover>*]:bg-hover',
		className
	)}
	{...restProps}
>
	{@render children?.()}
</tbody>
