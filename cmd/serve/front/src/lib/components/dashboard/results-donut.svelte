<!--
	Deployments by environment (LayerChart pie with a hole), the total in the middle and a legend
	with the numbers, so colour is never the only signal.
-->
<script lang="ts">
	import { PieChart } from 'layerchart';
	import * as Chart from '#lib/components/ui/chart/index.js';
	import l from '#lib/localization/index.ts';
	import type { Summary } from '#lib/resources/dashboard.ts';

	let { stats }: { stats: Summary['by_environment'] } = $props();

	const colors: Record<string, string> = {
		production: 'var(--color-primary)',
		staging: 'var(--color-faint)'
	};

	const config = {
		production: { label: l.translate('deploy.environment.production'), color: colors.production },
		staging: { label: l.translate('deploy.environment.staging'), color: colors.staging }
	} satisfies Chart.ChartConfig;

	const data = $derived(
		stats.map((s) => ({ key: s.environment, value: s.deployments, color: colors[s.environment] }))
	);
	const total = $derived(stats.reduce((sum, s) => sum + s.deployments, 0));
</script>

<figure
	class="m-0 flex flex-1 flex-col items-center justify-center gap-3"
	aria-label={stats.map((s) => `${config[s.environment].label} ${s.deployments}`).join(', ')}
>
	{#if total > 0}
		<div class="relative size-34">
			<Chart.Container {config} class="aspect-square size-34">
				<PieChart
					{data}
					key="key"
					value="value"
					c="color"
					innerRadius={-14}
					cornerRadius={3}
					padAngle={0.02}
				>
					{#snippet tooltip()}
						<Chart.Tooltip hideLabel />
					{/snippet}
				</PieChart>
			</Chart.Container>
			<span
				class="pointer-events-none absolute inset-0 flex flex-col items-center justify-center"
				aria-hidden="true"
			>
				<strong class="text-xl/7 font-semibold tabular-nums">{total}</strong>
				<span class="text-xs text-muted-foreground">{l.translate('dashboard.deploys.unit')}</span>
			</span>
		</div>
	{/if}
	<ul class="flex flex-wrap justify-center gap-x-4 gap-y-1 text-xs text-muted-foreground">
		{#each stats as s (s.environment)}
			<li class="flex items-center gap-1.5">
				<span class="size-2 rounded-full" style="background: {colors[s.environment]}"></span>
				{config[s.environment].label}
				<b class="font-semibold text-ink tabular-nums">{s.deployments}</b>
			</li>
		{:else}
			<li>{l.translate('dashboard.empty.range')}</li>
		{/each}
	</ul>
</figure>
