<!--
	What every day chart shares (DESIGN.md §11.10 Charts): one tab stop, the arrow keys move a
	highlight day by day, pointer or touch picks the nearest day, and the tooltip (floating style,
	flips left past the middle) is a live status. The drawing is hidden from screen readers; a
	hidden table holds every day and series instead.
-->
<script lang="ts">
	import type { Snippet } from 'svelte';
	import l from '#lib/localization/index.ts';
	import { longDay } from '#lib/charts.ts';

	type Props = {
		label: string;
		count: number;
		/** The highlighted day, -1 for none. */
		active: number;
		/** Width of the drawing, measured here. */
		width: number;
		/** x of the highlighted day, to place the tooltip. */
		anchor: number;
		/** Pointer x (from the left edge) to a day index. */
		pick: (x: number) => number;
		onpick?: (index: number) => void;
		/** The SVG drawing. */
		children: Snippet;
		tooltip: Snippet<[number]>;
		/** The hidden table: rows of [day, ...values]. */
		table: { headers: string[]; rows: { day: string; cells: string[] }[] };
	};

	let {
		label,
		count,
		active = $bindable(),
		width = $bindable(),
		anchor,
		pick,
		onpick,
		children,
		tooltip,
		table
	}: Props = $props();

	function move(event: PointerEvent) {
		const box = (event.currentTarget as HTMLElement).getBoundingClientRect();
		active = pick(event.clientX - box.left);
	}

	function key(event: KeyboardEvent) {
		if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight') return;
		event.preventDefault();
		const step = event.key === 'ArrowRight' ? 1 : -1;
		active = active < 0 ? count - 1 : Math.min(count - 1, Math.max(0, active + step));
	}

	const flip = $derived(anchor > width / 2);
</script>

<!-- A chart is one focusable region that reads its days with the arrow keys (its name says so); the
     numbers themselves are in the table below, so it stays a group rather than a widget. -->
<!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions -->
<div
	role="group"
	aria-label="{label}. {l.translate('charts.keys')}"
	tabindex="0"
	class="relative rounded-md outline-none focus-visible:ring-2 focus-visible:ring-ring"
	bind:clientWidth={width}
	onpointermove={move}
	onpointerdown={move}
	onpointerleave={() => (active = -1)}
	onblur={() => (active = -1)}
	onkeydown={key}
	onclick={() => active >= 0 && onpick?.(active)}
>
	{@render children()}
	<div
		role="status"
		class="pointer-events-none absolute top-1.5 z-10 min-w-40 rounded-md bg-popover px-2.5 py-2 text-xs/[18px] text-popover-foreground tabular-nums shadow-md"
		style:left={flip ? 'auto' : `${anchor + 12}px`}
		style:right={flip ? `${width - anchor + 12}px` : 'auto'}
		hidden={active < 0}
	>
		{#if active >= 0}{@render tooltip(active)}{/if}
	</div>
</div>
<!-- Wrapped: Firefox draws a caption of an sr-only table outside its clipped box. -->
<div class="sr-only">
	<table>
		<caption>{label}</caption>
		<thead>
			<tr>
				<th scope="col">{l.translate('charts.day')}</th>
				{#each table.headers as h (h)}<th scope="col">{h}</th>{/each}
			</tr>
		</thead>
		<tbody>
			{#each table.rows as row (row.day)}
				<tr>
					<th scope="row">{longDay(row.day)}</th>
					{#each row.cells as cell, i (i)}<td>{cell}</td>{/each}
				</tr>
			{/each}
		</tbody>
	</table>
</div>
