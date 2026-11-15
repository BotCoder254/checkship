<script lang="ts">
	import { cn, type WithElementRef } from '#lib/utils.js';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLTableRowElement>> = $props();
</script>

<tr
	bind:this={ref}
	data-slot="table-row"
	class={cn(
		// No lines between rows (DESIGN.md §5). Body rows fill on hover (see table-body).
		'transition-colors data-[state=selected]:*:bg-raised',
		className
	)}
	{...restProps}
>
	{@render children?.()}
</tr>
