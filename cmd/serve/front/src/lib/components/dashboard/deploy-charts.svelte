<!--
	The two charts side by side (DESIGN.md §11.10, owner 2026-11-17): Deployments per day on the
	left, Deploy time on the right, from a dashboard summary. Shared by the Overview and an
	application's Overview (the summary filtered by `app`).
-->
<script lang="ts">
	import ChartHistogramIcon from '@hugeicons/core-free-icons/ChartHistogramIcon';
	import Timer02Icon from '@hugeicons/core-free-icons/Timer02Icon';
	import ColumnChart from '#lib/components/charts/column-chart.svelte';
	import LineChart from '#lib/components/charts/line-chart.svelte';
	import Tile from '#lib/components/dashboard/tile.svelte';
	import EmptyState from '#lib/components/empty-state.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { niceSecondsCeil, type ChartSeries } from '#lib/charts.ts';
	import { formatSeconds } from '#lib/dashboard.ts';
	import l from '#lib/localization/index.ts';
	import type { DashboardRange, Day, Summary } from '#lib/resources/dashboard.ts';

	type Props = {
		summary: Summary | undefined;
		range: DashboardRange;
		error?: unknown;
		onretry?: () => void;
		/** A day with failures was clicked. */
		onfailures?: (day: Day) => void;
		/** Widen the range from the empty state. */
		onrange?: (range: DashboardRange) => void;
		/** Labels name the application ("Deployments of blog per day")... */
		app?: string;
		/** ...or the target ("Deployments to edge-01 per day"). */
		target?: string;
	};

	let { summary, range, error, onretry, onfailures, onrange, app, target }: Props = $props();

	const series = $derived(summary?.series ?? []);
	const loading = $derived(!summary && !error);

	const resultSeries: ChartSeries<Day>[] = [
		{ key: 'succeeded', label: l.translate('dashboard.series.succeeded'), color: 'var(--success)' },
		{ key: 'failed', label: l.translate('dashboard.series.failed'), color: 'var(--danger)' },
		{ key: 'other', label: l.translate('dashboard.series.other'), color: 'var(--faint)' }
	];
	const timeLines: (ChartSeries<Day> & { area?: boolean; dash?: boolean })[] = [
		{
			key: 'median_duration',
			label: l.translate('dashboard.duration.median'),
			color: 'var(--accent)',
			area: true
		},
		{
			key: 'max_duration',
			label: l.translate('dashboard.duration.slowest'),
			color: 'var(--warning)',
			dash: true
		}
	];
</script>

<div class="grid gap-x-10 gap-y-12 lg:grid-cols-2">
	<Tile
		title={l.translate('dashboard.deploys.title')}
		icon={ChartHistogramIcon}
		{loading}
		error={summary ? undefined : error}
		{onretry}
	>
		{#snippet actions()}{l.translate(
				onfailures ? 'dashboard.deploys.note' : 'dashboard.deploys.note.plain'
			)}{/snippet}
		{#if summary && summary.current.deployments + summary.current.other > 0}
			{@render legend(resultSeries)}
			<ColumnChart
				days={series}
				series={resultSeries}
				label={app
					? l.translate('dashboard.deploys.label.app', [app, range])
					: target
						? l.translate('dashboard.deploys.label.target', [target, range])
						: l.translate('dashboard.deploys.label', [range])}
				total={(n) => l.translate('charts.deployments', [n])}
				onselect={onfailures && ((day) => day.failed && onfailures(day))}
				hint={onfailures &&
					((day) => (day.failed ? l.translate('dashboard.deploys.hint_day') : undefined))}
			/>
		{:else}
			<EmptyState
				illustration="chart"
				size="compact"
				level={3}
				class="flex-1 justify-center"
				title={l.translate('dashboard.empty.range.title', [range])}
			>
				{#snippet action()}
					{#if range !== 90 && onrange}
						<Button variant="outline" size="sm" onclick={() => onrange(range === 7 ? 30 : 90)}>
							{l.translate('dashboard.range', [range === 7 ? 30 : 90])}
						</Button>
					{/if}
				{/snippet}
			</EmptyState>
		{/if}
	</Tile>

	<Tile
		title={l.translate('dashboard.time.title')}
		icon={Timer02Icon}
		{loading}
		error={summary ? undefined : error}
		{onretry}
	>
		{#snippet actions()}{l.translate('dashboard.time.note')}{/snippet}
		{#if series.some((d) => d.median_duration != null)}
			{@render legend(timeLines)}
			<LineChart
				days={series}
				lines={timeLines}
				label={app
					? l.translate('dashboard.time.label.app', [app, range])
					: target
						? l.translate('dashboard.time.label.target', [target, range])
						: l.translate('dashboard.time.label', [range])}
				format={formatSeconds}
				ceil={niceSecondsCeil}
				none={l.translate('dashboard.time.none')}
			/>
		{:else}
			<EmptyState
				illustration="chart"
				size="compact"
				level={3}
				class="flex-1 justify-center"
				title={l.translate('dashboard.duration.empty.title')}
				description={l.translate('dashboard.duration.empty')}
			/>
		{/if}
	</Tile>
</div>

{#snippet legend(items: { key: string; label: string; color: string; dash?: boolean }[])}
	<ul class="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
		{#each items as item (item.key)}
			<li class="flex items-center gap-1.5">
				{#if item.dash}<i class="w-3 border-t-2 border-dashed" style:border-color={item.color}
					></i>{:else}<i class="size-2.5 rounded-xs" style:background={item.color}></i>{/if}
				{item.label}
			</li>
		{/each}
	</ul>
{/snippet}
