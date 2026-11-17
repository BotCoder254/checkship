<!--
	The quiet axis every day chart shares: two dashed `line` gridlines at the rounded maximum and
	half of it, their value on a small surface patch so the line never runs through it, a solid
	`line-strong` baseline, and day labels (weekday letters, every 7th or 14th date, "Today" in the
	accent). Inside an SVG.
-->
<script lang="ts">
	import { tickIndexes, tickLabel } from '#lib/charts.ts';

	type Props = {
		width: number;
		top: number;
		plot: number;
		max: number;
		days: string[];
		/** x of the centre of day i. */
		x: (i: number) => number;
		format: (value: number) => string;
		/** The gridlines go behind the marks, the labels in front of them. */
		layer: 'grid' | 'labels';
	};

	let { width, top, plot, max, days, x, format, layer }: Props = $props();

	const y = (v: number) => top + plot - (v / max) * plot;
	const ticks = $derived(tickIndexes(days.length));
</script>

{#if layer === 'grid'}
	{#each [max, max / 2] as value (value)}
		<line
			x1="0"
			x2={width}
			y1={y(value)}
			y2={y(value)}
			class="stroke-line"
			stroke-dasharray="3 4"
		/>
	{/each}
	<line x1="0" x2={width} y1={top + plot + 0.5} y2={top + plot + 0.5} class="stroke-line-strong" />
{:else}
	{#each [max, max / 2] as value (value)}
		{@const text = format(value)}
		<rect
			x="0"
			y={y(value) - 8}
			width={text.length * 6.6 + 8}
			height="15"
			rx="3"
			class="fill-surface"
		/>
		<text x="4" y={y(value) + 3.5} class="fill-muted-foreground text-xs">{text}</text>
	{/each}
	{#each ticks as i (i)}
		{@const last = i === days.length - 1}
		<text
			x={Math.min(Math.max(x(i), 16), width - 18)}
			y={top + plot + 17}
			text-anchor="middle"
			class={last ? 'fill-primary text-xs font-semibold' : 'fill-muted-foreground text-xs'}
			>{tickLabel(days[i], i, days.length)}</text
		>
	{/each}
{/if}
