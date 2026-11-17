<!--
	Part of a whole (DESIGN.md §11.10 Charts): one 8px bar, a segment per item sized by its value,
	2px apart with 2px corners, read at a glance where a donut's angles are not. Under it a compact
	table with no visible header or lines: swatch and name on the left, right-aligned tabular figures
	(the first in ink, the rest muted). The header row is for screen readers only.
-->
<script lang="ts">
	import { cn } from '#lib/utils.js';

	export type ShareItem = {
		key: string;
		label: string;
		value: number;
		color: string;
		/** The figures after the name, first one in ink. */
		cells: string[];
		mono?: boolean;
	};

	type Props = { items: ShareItem[]; headers: string[]; label: string };

	let { items, headers, label }: Props = $props();

	const shown = $derived(items.filter((i) => i.value > 0));
</script>

<div class="flex flex-col gap-3">
	<div role="img" aria-label={label} class="flex h-2 gap-0.5">
		{#each shown as item (item.key)}
			<span
				class="h-full min-w-[3px] rounded-xs"
				style:flex-grow={item.value}
				style:background={item.color}
				title={item.label}
			></span>
		{/each}
	</div>
	<table class="w-full text-label tabular-nums">
		<thead class="sr-only">
			<tr
				>{#each headers as h (h)}<th scope="col">{h}</th>{/each}</tr
			>
		</thead>
		<tbody>
			{#each items as item (item.key)}
				<tr>
					<th scope="row" class="w-full max-w-0 py-1.5 text-left font-normal">
						<span class="flex min-w-0 items-center gap-2">
							<i class="size-2.5 shrink-0 rounded-xs" style:background={item.color}></i>
							<span class={cn('truncate', item.mono && 'font-mono text-xs')}>{item.label}</span>
						</span>
					</th>
					{#each item.cells as cell, i (i)}
						<td
							class={cn(
								'py-1.5 pl-4 text-right whitespace-nowrap',
								i > 0 && 'text-muted-foreground'
							)}>{cell}</td
						>
					{/each}
				</tr>
			{/each}
		</tbody>
	</table>
</div>
