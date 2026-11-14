<!--
	Median and slowest deployment duration per day (LayerChart area chart). Days without a finished
	deployment are left out rather than drawn as zero.
-->
<script lang="ts">
	import { AreaChart } from 'layerchart';
	import * as Chart from '#lib/components/ui/chart/index.js';
	import { formatSeconds } from '#lib/dashboard.ts';
	import l from '#lib/localization/index.ts';
	import type { Day } from '#lib/resources/dashboard.ts';

	let { days }: { days: Day[] } = $props();

	const config = {
		median: { label: l.translate('dashboard.duration.median'), color: 'var(--color-primary)' },
		max: { label: l.translate('dashboard.duration.slowest'), color: 'var(--color-warning)' }
	} satisfies Chart.ChartConfig;

	const format = new Intl.DateTimeFormat(l.locale(), { day: 'numeric', month: 'short' });

	const data = $derived(
		days
			.filter((d) => d.median_duration != null)
			.map((d) => ({
				date: new Date(`${d.date}T12:00:00`),
				median: d.median_duration ?? 0,
				max: d.max_duration ?? 0
			}))
	);
	const median = $derived(
		data.length
			? data.map((d) => d.median).sort((a, b) => a - b)[Math.floor(data.length / 2)]
			: null
	);
</script>

{#if data.length >= 2}
	<figure
		class="m-0 flex min-h-0 flex-1 flex-col"
		aria-label={l.translate('dashboard.duration.summary', [formatSeconds(median)])}
	>
		<Chart.Container {config} class="aspect-auto h-40 w-full">
			<AreaChart
				{data}
				x="date"
				series={[
					{
						key: 'max',
						label: config.max.label,
						color: config.max.color,
						props: { fillOpacity: 0.08 }
					},
					{
						key: 'median',
						label: config.median.label,
						color: config.median.color,
						props: { fillOpacity: 0.16 }
					}
				]}
				seriesLayout="overlap"
				grid={false}
				rule={false}
				padding={{ left: 40, bottom: 22, top: 6 }}
				props={{
					xAxis: {
						format: (d: Date) => format.format(d),
						ticks: 4,
						tickLabelProps: { class: 'fill-faint text-[11px]' }
					},
					yAxis: {
						format: (s: number) => (s < 60 ? `${Math.round(s)} s` : `${Math.round(s / 60)} min`),
						ticks: 3,
						tickLabelProps: { class: 'fill-faint text-[11px]' }
					}
				}}
			>
				{#snippet tooltip()}
					<Chart.Tooltip labelFormatter={(d: Date) => format.format(d)} />
				{/snippet}
			</AreaChart>
		</Chart.Container>
	</figure>
{:else}
	<p class="flex flex-1 items-center text-[13px] text-muted-foreground">
		{l.translate('dashboard.duration.empty')}
	</p>
{/if}
