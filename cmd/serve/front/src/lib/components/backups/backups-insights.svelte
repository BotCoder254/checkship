<!--
	The Backups page's figures and charts (DESIGN.md §11.10, Backups), counted in the browser from
	the backups list: four figures with a written comparison, backups per day on the left and space
	used on the right, then size by application, what started them, and a 13-week calendar. The
	range comes from the page header.
-->
<script lang="ts">
	import Archive02Icon from '@hugeicons/core-free-icons/Archive02Icon';
	import Calendar03Icon from '@hugeicons/core-free-icons/Calendar03Icon';
	import DashboardSquare01Icon from '@hugeicons/core-free-icons/DashboardSquare01Icon';
	import HardDriveIcon from '@hugeicons/core-free-icons/HardDriveIcon';
	import Tag01Icon from '@hugeicons/core-free-icons/Tag01Icon';
	import CalendarHeatmap from '#lib/components/charts/calendar-heatmap.svelte';
	import ColumnChart from '#lib/components/charts/column-chart.svelte';
	import LineChart from '#lib/components/charts/line-chart.svelte';
	import Metric from '#lib/components/charts/metric.svelte';
	import ShareBar, { type ShareItem } from '#lib/components/charts/share-bar.svelte';
	import Tile from '#lib/components/dashboard/tile.svelte';
	import EmptyState from '#lib/components/empty-state.svelte';
	import { appName, formatBytes, triggerLabel } from '#lib/backup.ts';
	import {
		backupDays,
		backupFigures,
		byTrigger,
		sizeByApp,
		type BackupDay
	} from '#lib/backup-stats.ts';
	import { countChange, niceBytesCeil, rateChange, type ChartSeries } from '#lib/charts.ts';
	import { formatRate } from '#lib/dashboard.ts';
	import l from '#lib/localization/index.ts';
	import type { Snippet } from 'svelte';
	import type { Backup } from '#lib/resources/backups.ts';
	import { cn } from '#lib/utils.js';

	type Props = {
		backups: Backup[];
		range: number;
		/** Show size by application (off on one application's tab). */
		byApp?: boolean;
		/** Right after the figures: Freshness on the Backups page. */
		afterFigures?: Snippet;
	};

	let { backups, range, byApp = true, afterFigures }: Props = $props();

	const days = $derived(backupDays(backups, range));
	const calendar = $derived(backupDays(backups, 91));
	const figures = $derived(backupFigures(backups, range));
	const space = $derived(formatBytes(figures.stored + figures.local).split(' '));
	const hasRange = $derived(days.some((d) => d.succeeded + d.failed + d.other > 0));

	const shades = [
		'var(--accent)',
		'color-mix(in oklab, var(--accent) 55%, var(--ink))',
		'color-mix(in oklab, var(--accent) 35%, var(--surface))',
		'var(--faint)',
		'var(--line-strong)'
	];
	const shade = (i: number) => shades[Math.min(i, shades.length - 1)];
	const percent = (part: number, whole: number) =>
		`${whole ? Math.round((part / whole) * 100) : 0}%`;

	const resultSeries: ChartSeries<BackupDay>[] = [
		{ key: 'succeeded', label: l.translate('dashboard.series.succeeded'), color: 'var(--success)' },
		{ key: 'failed', label: l.translate('dashboard.series.failed'), color: 'var(--danger)' },
		{ key: 'other', label: l.translate('dashboard.series.other'), color: 'var(--faint)' }
	];
	const spaceSeries: ChartSeries<BackupDay>[] = [
		{ key: 'stored', label: l.translate('backups.space.stored'), color: 'var(--accent)' },
		{
			key: 'local',
			label: l.translate('backups.space.local'),
			color: 'color-mix(in oklab, var(--accent) 50%, var(--ink))'
		}
	];

	const apps = $derived.by((): ShareItem[] => {
		const all = sizeByApp(backups);
		const total = all.reduce((sum, a) => sum + a.size, 0);
		return all.map((a, i) => ({
			key: a.id,
			label: a.deleted ? l.translate('backups.deleted_app', [a.name]) : a.name,
			value: a.size,
			color: shade(i),
			cells: [formatBytes(a.size), percent(a.size, total), l.translate('charts.backups', [a.count])]
		}));
	});

	const triggers = $derived.by((): ShareItem[] => {
		const all = byTrigger(backups, range);
		const total = all.reduce((sum, t) => sum + t.count, 0);
		return all.map((t, i) => ({
			key: t.trigger,
			label: triggerLabel(t.trigger),
			value: t.count,
			color: shade(i),
			cells: [String(t.count), percent(t.count, total)]
		}));
	});
</script>

<div class="flex flex-col gap-12">
	<section
		aria-label={l.translate('backups.figures.label', [range])}
		class="grid grid-cols-2 gap-6 track p-4 sm:gap-x-10 sm:p-5 xl:grid-cols-4"
	>
		<Metric
			label={l.translate('backups.kpi.count')}
			value={String(figures.count)}
			change={countChange(figures.count, figures.previousCount, range)}
		/>
		<Metric
			label={l.translate('dashboard.kpi.success')}
			value={formatRate(figures.rate)}
			unit={figures.rate != null ? '%' : ''}
			change={rateChange(figures.rate, figures.previousRate, range)}
		/>
		<Metric
			label={l.translate('backups.kpi.space')}
			value={space[0]}
			unit={space[1]}
			change={l.translate('backups.kpi.space.split', [
				formatBytes(figures.stored),
				formatBytes(figures.local)
			])}
		/>
		<Metric
			label={l.translate('backups.kpi.last')}
			value={figures.lastGood ? l.relative(figures.lastGood.requested_at) : '—'}
			change={figures.lastGood
				? `${appName(figures.lastGood)} · ${l.translate(`deploy.environment.${figures.lastGood.environment}`)} · ${formatBytes(figures.lastGood.result?.size ?? 0)}`
				: l.translate('backups.kpi.last.none')}
		/>
	</section>

	{@render afterFigures?.()}

	<div class="grid gap-x-10 gap-y-12 lg:grid-cols-2">
		<Tile title={l.translate('backups.chart.count')} icon={Archive02Icon}>
			{#snippet actions()}{l.translate('backups.chart.count.note')}{/snippet}
			{#if hasRange}
				{@render legend(resultSeries)}
				<ColumnChart
					{days}
					series={resultSeries}
					label={l.translate('backups.chart.count.label', [range])}
					total={(n) => l.translate('charts.backups', [n])}
				/>
			{:else}
				{@render empty()}
			{/if}
		</Tile>
		<Tile title={l.translate('backups.chart.space')} icon={HardDriveIcon}>
			{#snippet actions()}{l.translate('backups.chart.space.note')}{/snippet}
			{#if figures.stored + figures.local > 0}
				{@render legend(spaceSeries)}
				<LineChart
					{days}
					lines={spaceSeries}
					stack
					label={l.translate('backups.chart.space.label', [range])}
					format={formatBytes}
					ceil={niceBytesCeil}
				/>
			{:else}
				{@render empty()}
			{/if}
		</Tile>
	</div>

	<div class={cn('grid gap-x-10 gap-y-12 md:grid-cols-2', byApp && 'xl:grid-cols-3')}>
		{#if byApp}
			<Tile title={l.translate('backups.share.apps')} icon={DashboardSquare01Icon}>
				{#snippet actions()}{l.translate('backups.share.apps.note')}{/snippet}
				<ShareBar
					items={apps}
					headers={[
						l.translate('column.application'),
						l.translate('backups.share.size'),
						l.translate('charts.share'),
						l.translate('backups.kpi.count')
					]}
					label={apps.map((a) => `${a.label} ${a.cells[0]}`).join(', ')}
				/>
			</Tile>
		{/if}
		<Tile title={l.translate('backups.share.triggers')} icon={Tag01Icon}>
			{#snippet actions()}{l.translate('charts.last_days', [range])}{/snippet}
			{#if triggers.length}
				<ShareBar
					items={triggers}
					headers={[
						l.translate('column.trigger'),
						l.translate('backups.kpi.count'),
						l.translate('charts.share')
					]}
					label={triggers.map((t) => `${t.label} ${t.value}`).join(', ')}
				/>
			{:else}
				{@render empty()}
			{/if}
		</Tile>
		<Tile
			title={l.translate('backups.calendar')}
			icon={Calendar03Icon}
			class={byApp ? 'md:col-span-2 xl:col-span-1' : undefined}
		>
			{#snippet actions()}{l.translate('dashboard.calendar.note')}{/snippet}
			<CalendarHeatmap
				days={calendar.map((d) => ({ date: d.date, value: d.succeeded }))}
				label={l.translate('backups.calendar.label')}
				count={(n) => l.translate('charts.backups', [n])}
			/>
		</Tile>
	</div>
</div>

{#snippet legend(items: { key: string; label: string; color: string }[])}
	<ul class="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
		{#each items as item (item.key)}
			<li class="flex items-center gap-1.5">
				<i class="size-2.5 rounded-xs" style:background={item.color}></i>{item.label}
			</li>
		{/each}
	</ul>
{/snippet}

{#snippet empty()}
	<EmptyState
		illustration="chart"
		size="compact"
		level={3}
		class="flex-1 justify-center"
		title={l.translate('backups.chart.empty', [range])}
	/>
{/snippet}
