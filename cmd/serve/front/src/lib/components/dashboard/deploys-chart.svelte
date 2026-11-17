<!--
	Deployments per day, stacked by result (LayerChart through shadcn-svelte Chart). Colours are
	palette tokens; the legend and tooltip name each series, so colour is never the only signal.
	Screen readers get a one-line summary and the numbers as a table.
-->
<script lang="ts">
	import { BarChart } from 'layerchart';
	import * as Chart from '#lib/components/ui/chart/index.js';
	import l from '#lib/localization/index.ts';
	import type { Day } from '#lib/resources/dashboard.ts';

	type Props = {
		days: Day[];
		onselect?: (day: Day) => void;
	};

	let { days, onselect }: Props = $props();

	const config = {
		succeeded: { label: l.translate('dashboard.series.succeeded'), color: 'var(--color-success)' },
		failed: { label: l.translate('dashboard.series.failed'), color: 'var(--color-danger)' },
		other: { label: l.translate('dashboard.series.other'), color: 'var(--color-faint)' }
	} satisfies Chart.ChartConfig;

	const format = new Intl.DateTimeFormat(l.locale(), { day: 'numeric', month: 'short' });
	const label = (date: string) => format.format(new Date(`${date}T12:00:00`));

	// About five labels whatever the range.
	const ticks = $derived.by(() => {
		const step = Math.max(1, Math.ceil(days.length / 5));
		// Offset by half a step so the first label clears the y axis.
		return days.filter((_, i) => i % step === Math.floor(step / 2)).map((d) => d.date);
	});

	const totals = $derived(
		days.reduce(
			(acc, d) => ({
				succeeded: acc.succeeded + d.succeeded,
				failed: acc.failed + d.failed,
				other: acc.other + d.other
			}),
			{ succeeded: 0, failed: 0, other: 0 }
		)
	);
</script>

<figure
	class="relative m-0 flex min-h-0 flex-1 flex-col"
	aria-label={l.translate('dashboard.deploys.summary', [
		days.length,
		totals.succeeded,
		totals.failed,
		totals.other
	])}
>
	<Chart.Container {config} class="aspect-auto h-full min-h-60 w-full flex-1">
		<BarChart
			data={days}
			x="date"
			series={[
				{ key: 'succeeded', label: config.succeeded.label, color: config.succeeded.color },
				{ key: 'failed', label: config.failed.label, color: config.failed.color },
				{ key: 'other', label: config.other.label, color: config.other.color }
			]}
			seriesLayout="stack"
			bandPadding={0.3}
			grid={false}
			rule={false}
			padding={{ left: 28, bottom: 22, top: 6 }}
			props={{
				xAxis: { format: label, ticks, tickLabelProps: { class: 'fill-muted-foreground text-xs' } },
				yAxis: { ticks: 3, tickLabelProps: { class: 'fill-muted-foreground text-xs' } },
				bars: { stroke: 'none', radius: 2 }
			}}
			onBarClick={(_, detail) => onselect?.(detail.data as Day)}
		>
			{#snippet tooltip()}
				<Chart.Tooltip labelFormatter={(value: string) => label(value)} />
			{/snippet}
		</BarChart>
	</Chart.Container>
	<table class="sr-only">
		<caption>{l.translate('dashboard.deploys.title')}</caption>
		<thead>
			<tr>
				<th scope="col">{l.translate('dashboard.day')}</th>
				<th scope="col">{config.succeeded.label}</th>
				<th scope="col">{config.failed.label}</th>
				<th scope="col">{config.other.label}</th>
			</tr>
		</thead>
		<tbody>
			{#each days as day (day.date)}
				<tr>
					<th scope="row">{label(day.date)}</th>
					<td>{day.succeeded}</td>
					<td>{day.failed}</td>
					<td>{day.other}</td>
				</tr>
			{/each}
		</tbody>
	</table>
</figure>
