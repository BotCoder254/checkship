<!--
	Stats strip (DESIGN.md §11.10): the period's four figures in one sunken band across the grid,
	not four cards. Segments are separated by space, never by lines (no dividers). Each segment: a
	label with a small icon, the figure, its change against the previous period (green when it is
	good news, the arrow gives the direction), the previous value in words and a sparkline.
	Four across from xl, two by two below; phones drop the previous-value line.
-->
<script lang="ts">
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import ArrowDownRight01Icon from '@hugeicons/core-free-icons/ArrowDownRight01Icon';
	import ArrowUpRight01Icon from '@hugeicons/core-free-icons/ArrowUpRight01Icon';
	import CheckmarkCircle02Icon from '@hugeicons/core-free-icons/CheckmarkCircle02Icon';
	import RefreshIcon from '@hugeicons/core-free-icons/RefreshIcon';
	import Rocket01Icon from '@hugeicons/core-free-icons/Rocket01Icon';
	import Timer02Icon from '@hugeicons/core-free-icons/Timer02Icon';
	import Sparkline from '#lib/components/dashboard/sparkline.svelte';
	import Icon, { type IconData } from '#lib/components/icon.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Skeleton } from '#lib/components/ui/skeleton/index.js';
	import {
		countTrend,
		durationTrend,
		formatRate,
		formatSeconds,
		rateTrend,
		splitSeconds,
		type Trend
	} from '#lib/dashboard.ts';
	import l from '#lib/localization/index.ts';
	import type { Summary } from '#lib/resources/dashboard.ts';
	import { cn } from '#lib/utils.js';

	type Props = {
		summary: Summary | undefined;
		range: number;
		error?: unknown;
		onretry?: () => void;
		class?: string;
	};

	let { summary, range, error, onretry, class: className }: Props = $props();

	type Segment = {
		key: string;
		label: string;
		icon: IconData;
		value: string;
		unit: string;
		trend: Trend | null;
		previous: string;
		spark: (number | null)[];
		tone: 'accent' | 'success' | 'warning' | 'danger';
	};

	const segments = $derived.by((): Segment[] => {
		if (!summary) return [];
		const { current, previous, series } = summary;
		const hasPrevious = previous.deployments > 0;
		const none = l.translate('dashboard.strip.first');
		const [median, medianUnit] = splitSeconds(current.median_duration);
		const [recover, recoverUnit] = splitSeconds(current.median_recovery);
		const environment = l.translate(`deploy.environment.${summary.recovery.environment}`);

		return [
			{
				key: 'deployments',
				label: l.translate('dashboard.kpi.deployments'),
				icon: Rocket01Icon,
				value: String(current.deployments),
				unit: '',
				trend: countTrend(current.deployments, previous.deployments),
				previous: hasPrevious
					? l.translate('dashboard.strip.previous_count', [previous.deployments, range])
					: none,
				spark: series.map((d) => d.succeeded + d.failed + d.other),
				tone: 'accent'
			},
			{
				key: 'success',
				label: l.translate('dashboard.kpi.success'),
				icon: CheckmarkCircle02Icon,
				value: formatRate(current.success_rate),
				unit: current.success_rate != null ? '%' : '',
				trend: rateTrend(current.success_rate, previous.success_rate),
				previous:
					previous.success_rate != null
						? l.translate('dashboard.strip.before', [`${formatRate(previous.success_rate)}%`])
						: none,
				spark: series.map((d) =>
					d.succeeded + d.failed ? d.succeeded / (d.succeeded + d.failed) : null
				),
				tone: 'success'
			},
			{
				key: 'duration',
				label: l.translate('dashboard.kpi.duration'),
				icon: Timer02Icon,
				value: median,
				unit: medianUnit,
				trend: durationTrend(current.median_duration, previous.median_duration),
				previous:
					previous.median_duration != null
						? l.translate('dashboard.strip.before', [formatSeconds(previous.median_duration)])
						: none,
				spark: series.map((d) => d.median_duration),
				tone: 'warning'
			},
			{
				key: 'recover',
				label: l.translate('dashboard.kpi.recover'),
				icon: RefreshIcon,
				value: recover,
				unit: recoverUnit,
				trend: durationTrend(current.median_recovery, previous.median_recovery),
				previous: current.recoveries
					? previous.median_recovery != null
						? l.translate('dashboard.strip.recover', [
								formatSeconds(previous.median_recovery),
								current.recoveries,
								environment
							])
						: l.translate('dashboard.strip.recoveries', [current.recoveries, environment])
					: l.translate('dashboard.kpi.recover.none'),
				spark: series.map((d) => d.failed),
				tone: 'danger'
			}
		];
	});
</script>

<section
	aria-label={l.translate('dashboard.strip.label', [range])}
	class={cn(
		'grid grid-cols-2 gap-x-4 gap-y-5 rounded-lg bg-sunken p-4 sm:gap-x-10 sm:gap-y-6 sm:px-7 sm:py-5.5 xl:grid-cols-4',
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
			<div class="flex flex-col gap-2.5" aria-busy="true">
				<Skeleton class="h-3 w-1/2 bg-raised" />
				<Skeleton class="h-7.5 w-3/4 bg-raised" />
				<Skeleton class="h-7 w-full bg-raised" />
			</div>
		{/each}
	{:else}
		{#each segments as s (s.key)}
			<div
				role="group"
				aria-labelledby="stat-{s.key}"
				class="flex min-w-0 flex-col gap-1.5 sm:gap-2"
			>
				<span
					id="stat-{s.key}"
					class="flex items-center gap-1.75 text-[12.5px] text-muted-foreground sm:text-[13px]"
				>
					<Icon icon={s.icon} class="hidden size-3.75 text-faint sm:block" />
					{s.label}
				</span>
				<span class="flex flex-wrap items-baseline gap-x-2.5 gap-y-1.5">
					<strong
						class="text-2xl/none font-semibold tracking-tight tabular-nums sm:text-[32px]/none"
					>
						{s.value}{#if s.unit}<span
								class="ml-0.5 text-[13px] font-medium text-muted-foreground sm:text-base"
								>{s.unit}</span
							>{/if}
					</strong>
					{#if s.trend}
						<span
							class={cn(
								'inline-flex h-5.5 items-center gap-1 rounded-sm px-1.5 text-xs font-semibold',
								s.trend.good ? 'bg-success-soft text-success' : 'bg-danger-soft text-danger'
							)}
						>
							<Icon
								icon={s.trend.up ? ArrowUpRight01Icon : ArrowDownRight01Icon}
								class="size-3.5"
							/>
							{s.trend.text}
						</span>
					{/if}
				</span>
				<span class="hidden truncate text-xs text-faint sm:block">{s.previous}</span>
				<div class="mt-0.5 h-7 sm:h-8"><Sparkline values={s.spark} tone={s.tone} /></div>
			</div>
		{/each}
	{/if}
</section>
