<!--
	Stacked columns, one per day (DESIGN.md §11.10 Charts): for counts that come in bursts
	(deployments, backups), so no value is invented between days. 3px between days (1px past 60),
	1px between the layers of a day, 2px corners on the top segment only; a day with nothing is a
	1px hairline so it reads as zero. Hover fades the other days to 60%. `onselect` makes a day
	clickable (a day with failures opens them).
-->
<script lang="ts" generics="Day extends { date: string }">
	import { longDay, niceCeil, type ChartSeries } from '#lib/charts.ts';
	import l from '#lib/localization/index.ts';
	import ChartAxis from './chart-axis.svelte';
	import ChartFrame from './chart-frame.svelte';

	type Props = {
		days: Day[];
		series: ChartSeries<Day>[];
		label: string;
		/** "5 deployments" */
		total: (n: number) => string;
		onselect?: (day: Day) => void;
		/** Shown in the tooltip of a day that can be selected. */
		hint?: (day: Day) => string | undefined;
		height?: number;
	};

	let { days, series, label, total, onselect, hint, height = 200 }: Props = $props();

	let width = $state(0);
	let active = $state(-1);

	const TOP = 6;
	const BOTTOM = 22;
	const plot = $derived(height - TOP - BOTTOM);
	const n = $derived(days.length);
	const gap = $derived(n > 60 ? 1 : 3);
	const cw = $derived(n ? Math.max(1, (width - gap * (n - 1)) / n) : 0);
	const value = (d: Day, key: string) => Number(d[key as keyof Day]) || 0;
	const sums = $derived(days.map((d) => series.reduce((s, k) => s + value(d, k.key), 0)));
	// Counts: at least 2, so the half gridline is a whole number.
	const max = $derived(niceCeil(Math.max(2, ...sums)));
	const x = (i: number) => i * (cw + gap) + cw / 2;
	const y = (v: number) => TOP + plot - (v / max) * plot;

	/** The layers of day i, bottom up, with the top one marked. */
	function layers(i: number) {
		const d = days[i];
		const shown = series.filter((k) => value(d, k.key) > 0);
		let acc = 0;
		return shown.map((k, j) => {
			const v = value(d, k.key);
			const top = y(acc + v);
			const last = j === shown.length - 1;
			const h = Math.max(1, (v / max) * plot - (last ? 0 : 1));
			acc += v;
			return { key: k.key, color: k.color, top, h, last };
		});
	}

	function topPath(x0: number, top: number, w: number, h: number) {
		const r = Math.min(2, w / 2, h);
		return `M${x0} ${top + h}V${top + r}Q${x0} ${top} ${x0 + r} ${top}H${x0 + w - r}Q${x0 + w} ${top} ${x0 + w} ${top + r}V${top + h}Z`;
	}
</script>

<ChartFrame
	{label}
	count={n}
	bind:active
	bind:width
	anchor={active >= 0 ? x(active) : 0}
	pick={(px) => Math.min(n - 1, Math.max(0, Math.floor((px / Math.max(1, width)) * n)))}
	onpick={(i) => onselect?.(days[i])}
	table={{
		headers: series.map((s) => s.label),
		rows: days.map((d) => ({ day: d.date, cells: series.map((s) => String(value(d, s.key))) }))
	}}
>
	<svg
		aria-hidden="true"
		{width}
		{height}
		viewBox="0 0 {width} {height}"
		class="block overflow-visible"
	>
		{#if width > 0}
			<ChartAxis
				{width}
				top={TOP}
				{plot}
				{max}
				days={days.map((d) => d.date)}
				{x}
				format={(v) => String(v)}
				layer="grid"
			/>
			{#each days as d, i (d.date)}
				{@const x0 = i * (cw + gap)}
				<g opacity={active >= 0 && active !== i ? 0.6 : 1}>
					{#if sums[i] === 0}
						<rect x={x0} y={TOP + plot - 1} width={cw} height="1" class="fill-line-strong" />
					{:else}
						{#each layers(i) as layer (layer.key)}
							{#if layer.last && layer.h > 3}
								<path d={topPath(x0, layer.top, cw, layer.h)} style:fill={layer.color} />
							{:else}
								<rect x={x0} y={layer.top} width={cw} height={layer.h} style:fill={layer.color} />
							{/if}
						{/each}
					{/if}
				</g>
			{/each}
			<ChartAxis
				{width}
				top={TOP}
				{plot}
				{max}
				days={days.map((d) => d.date)}
				{x}
				format={(v) => String(v)}
				layer="labels"
			/>
		{/if}
	</svg>
	{#snippet tooltip(i)}
		{@const d = days[i]}
		<b class="block text-label font-semibold">{longDay(d.date)}</b>
		<span class="flex justify-between gap-4"
			><span class="text-muted-foreground">{l.translate('charts.total')}</span><b
				class="font-semibold">{total(sums[i])}</b
			></span
		>
		{#each series as s (s.key)}
			<span class="flex justify-between gap-4">
				<span class="flex items-center gap-1.5 text-muted-foreground"
					><i class="size-2.5 rounded-xs" style:background={s.color}></i>{s.label}</span
				>
				{value(d, s.key)}
			</span>
		{/each}
		{#if hint?.(d)}<span class="mt-1 block text-muted-foreground">{hint(d)}</span>{/if}
	{/snippet}
</ChartFrame>
