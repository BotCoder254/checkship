<!--
	The Overview's four figures (DESIGN.md §11.10 Figures): Deployments, Success rate, Median deploy
	time and Time to recover, each a big number with one line comparing it with the previous period
	of the same length in words. In one row on the page from xl, two by two below; no band, no cards,
	no sparklines (the charts under them show the trend). Skeletons shaped like the figures while
	loading; a failed load says so and offers Try again.
-->
<script lang="ts">
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import RefreshIcon from '@hugeicons/core-free-icons/RefreshIcon';
	import Metric from '#lib/components/charts/metric.svelte';
	import Icon from '#lib/components/icon.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import { countChange, durationChange, rateChange } from '#lib/charts.ts';
	import { formatRate, splitSeconds } from '#lib/dashboard.ts';
	import l from '#lib/localization/index.ts';
	import type { Summary } from '#lib/resources/dashboard.ts';
	import { cn } from '#lib/utils.js';

	type Props = {
		summary: Summary | undefined;
		range: number;
		error?: unknown;
		onretry?: () => void;
		/** The region's name (an application's page names it after the app). */
		label?: string;
		class?: string;
	};

	let { summary, range, error, onretry, label, class: className }: Props = $props();

	const figures = $derived.by(() => {
		if (!summary) return [];
		const { current, previous } = summary;
		const [median, medianUnit] = splitSeconds(current.median_duration);
		const [recover, recoverUnit] = splitSeconds(current.median_recovery);
		return [
			{
				key: 'deployments',
				label: l.translate('dashboard.kpi.deployments'),
				value: String(current.deployments),
				unit: '',
				change: countChange(current.deployments, previous.deployments, range)
			},
			{
				key: 'success',
				label: l.translate('dashboard.kpi.success'),
				value: formatRate(current.success_rate),
				unit: current.success_rate != null ? '%' : '',
				change: rateChange(current.success_rate, previous.success_rate, range)
			},
			{
				key: 'duration',
				label: l.translate('dashboard.kpi.duration'),
				value: median,
				unit: medianUnit,
				change: durationChange(current.median_duration, previous.median_duration, range)
			},
			{
				key: 'recover',
				label: l.translate('dashboard.kpi.recover'),
				value: recover,
				unit: recoverUnit,
				change: current.recoveries
					? durationChange(current.median_recovery, previous.median_recovery, range)
					: l.translate('dashboard.kpi.recover.none')
			}
		];
	});
</script>

<section
	aria-label={label ?? l.translate('dashboard.strip.label', [range])}
	class={cn(
		'grid grid-cols-2 gap-x-6 gap-y-6 track p-4 sm:gap-x-10 sm:p-5 xl:grid-cols-4',
		className
	)}
>
	{#if error && !summary}
		<div class="col-span-full flex flex-wrap items-center gap-3 py-2">
			<span class="flex items-center gap-2 text-sm text-danger">
				<Icon icon={Alert02Icon} class="size-4" />
				{l.translate('dashboard.tile.error')}
			</span>
			{#if onretry}
				<Button variant="outline" size="sm" onclick={onretry}>
					<Icon icon={RefreshIcon} />
					{l.translate('dashboard.retry')}
				</Button>
			{/if}
		</div>
	{:else if !summary}
		{#each [0, 1, 2, 3] as i (i)}
			<div class="flex flex-col gap-2" aria-busy="true">
				<Skeleton class="h-3.5 w-1/2 bg-raised" />
				<Skeleton class="h-9 w-2/5 bg-raised" />
				<Skeleton class="h-3.5 w-4/5 bg-raised" />
			</div>
		{/each}
	{:else}
		{#each figures as f (f.key)}
			<Metric label={f.label} value={f.value} unit={f.unit} change={f.change} />
		{/each}
	{/if}
</section>
