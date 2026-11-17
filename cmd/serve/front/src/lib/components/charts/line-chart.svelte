<!--
	Lines over days (DESIGN.md §11.10 Charts): for a measurement that moves (deploy time) or an
	amount that builds up (space used). Monotone curves never overshoot a day's value; a day with no
	value leaves a gap (a lone day is a dot). `stack` draws filled layers on top of each other
	(88% fill, 1.5px top line); otherwise a 2px line with an optional soft area, or a dashed line.
	Hover draws a guide line and a ring on the point.
-->
<script lang="ts" generics="Day extends { date: string }">
	import { longDay, monotonePath, niceCeil, runs, type Point } from '#lib/charts.ts';
	import l from '#lib/localization/index.ts';
	import ChartAxis from './chart-axis.svelte';
	import ChartFrame from './chart-frame.svelte';

	type Line = {
		key: keyof Day & string;
		label: string;
		color: string;
		/** A soft area under the line. */
		area?: boolean;
		dash?: boolean;
	};

	type Props = {
		days: Day[];
		lines: Line[];
		label: string;
		format: (value: number) => string;
		/** Stack the lines as filled layers (a total built from parts). */
		stack?: boolean;
		/** The top of the scale from the highest value (niceCeil by default; bytes and seconds have their own). */
		ceil?: (value: number) => number;
		/** Tooltip text for a day without a value. */
		none?: string;
		height?: number;
	};

	let {
		days,
		lines,
		label,
		format,
		stack = false,
		ceil = niceCeil,
		none = '—',
		height = 200
	}: Props = $props();

	let width = $state(0);
	let active = $state(-1);

	const TOP = 6;
	const BOTTOM = 22;
	const plot = $derived(height - TOP - BOTTOM);
	const n = $derived(days.length);
	const get = (d: Day, key: string): number | null => {
		const v = d[key as keyof Day];
		return v == null ? null : Number(v);
	};
	const totals = $derived(days.map((d) => lines.reduce((s, k) => s + (get(d, k.key) ?? 0), 0)));
	const max = $derived(
		ceil(
			Math.max(1, ...(stack ? totals : days.flatMap((d) => lines.map((k) => get(d, k.key) ?? 0))))
		)
	);
	const x = (i: number) => (n <= 1 ? width / 2 : 4 + (i * (width - 8)) / (n - 1));
	const y = (v: number) => TOP + plot - (v / max) * plot;

	/** Stacked layers: the top edge of each and the polygon down to the one below. */
	const layers = $derived.by(() => {
		if (!stack) return [];
		let base = days.map(() => 0);
		return lines.map((k) => {
			const top = days.map((d, i) => base[i] + (get(d, k.key) ?? 0));
			const upper: Point[] = top.map((v, i) => [x(i), y(v)]);
			const lower: Point[] = base.map((v, i) => [x(i), y(v)] as Point).reverse();
			base = top;
			return {
				key: k.key,
				color: k.color,
				edge: monotonePath(upper),
				fill: `${monotonePath(upper)}L${monotonePath(lower).slice(1)}Z`
			};
		});
	});

	const ring = $derived.by(() => {
		if (active < 0) return undefined;
		const v = stack ? totals[active] : get(days[active], lines[0].key);
		return v == null ? undefined : { y: y(v), color: lines[stack ? lines.length - 1 : 0].color };
	});
</script>

<ChartFrame
	{label}
	count={n}
	bind:active
	bind:width
	anchor={active >= 0 ? x(active) : 0}
	pick={(px) =>
		Math.min(n - 1, Math.max(0, Math.round(((px - 4) / Math.max(1, width - 8)) * (n - 1))))}
	table={{
		headers: lines.map((k) => k.label),
		rows: days.map((d) => ({
			day: d.date,
			cells: lines.map((k) => {
				const v = get(d, k.key);
				return v == null ? none : format(v);
			})
		}))
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
				{format}
				layer="grid"
			/>
			{#if stack}
				{#each layers as layer (layer.key)}
					<path d={layer.fill} style:fill={layer.color} opacity="0.88" />
					<path d={layer.edge} fill="none" style:stroke={layer.color} stroke-width="1.5" />
				{/each}
			{:else}
				{#each lines as k (k.key)}
					{#each runs(days.map((d) => get(d, k.key))) as run (run.start)}
						{@const points = run.values.map((v, j) => [x(run.start + j), y(v)] as Point)}
						{#if points.length === 1}
							<circle cx={points[0][0]} cy={points[0][1]} r="2.5" style:fill={k.color} />
						{:else}
							{#if k.area}
								<path
									d="{monotonePath(points)}L{points.at(-1)![0]} {y(0)}L{points[0][0]} {y(0)}Z"
									style:fill="color-mix(in oklab, {k.color} 14%, transparent)"
								/>
							{/if}
							<path
								d={monotonePath(points)}
								fill="none"
								style:stroke={k.color}
								stroke-width={k.dash ? 1.5 : 2}
								stroke-dasharray={k.dash ? '4 4' : undefined}
								stroke-linecap="round"
							/>
						{/if}
					{/each}
				{/each}
			{/if}
			{#if active >= 0}
				<line x1={x(active)} x2={x(active)} y1={TOP} y2={TOP + plot} class="stroke-line-strong" />
				{#if ring}
					<circle
						cx={x(active)}
						cy={ring.y}
						r="4.5"
						class="fill-surface"
						style:stroke={ring.color}
						stroke-width="2"
					/>
				{/if}
			{/if}
			<ChartAxis
				{width}
				top={TOP}
				{plot}
				{max}
				days={days.map((d) => d.date)}
				{x}
				{format}
				layer="labels"
			/>
		{/if}
	</svg>
	{#snippet tooltip(i)}
		{@const d = days[i]}
		<b class="block text-label font-semibold">{longDay(d.date)}</b>
		{#if stack}
			<span class="flex justify-between gap-4"
				><span class="text-muted-foreground">{l.translate('charts.total')}</span><b
					class="font-semibold">{format(totals[i])}</b
				></span
			>
		{/if}
		{#each lines as k (k.key)}
			{@const v = get(d, k.key)}
			<span class="flex justify-between gap-4">
				<span class="flex items-center gap-1.5 text-muted-foreground">
					{#if k.dash}<i class="w-3 border-t-2 border-dashed" style:border-color={k.color}
						></i>{:else}<i class="size-2.5 rounded-xs" style:background={k.color}></i>{/if}{k.label}
				</span>
				{v == null ? none : format(v)}
			</span>
		{/each}
	{/snippet}
</ChartFrame>
