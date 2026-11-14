<!--
	A small trend line under a KPI (decorative: the figure and its change carry the meaning).
	Null values are skipped. `tone` is a palette token: accent, success, warning or danger.
-->
<script lang="ts">
	type Props = { values: (number | null)[]; tone: 'accent' | 'success' | 'warning' | 'danger' };

	let { values, tone }: Props = $props();

	const W = 220;
	const H = 40;

	const path = $derived.by(() => {
		const points = values
			.map((v, i) => (v == null ? null : ([i, v] as const)))
			.filter((p) => p !== null);
		if (points.length < 2) return undefined;
		const ys = points.map(([, v]) => v);
		const min = Math.min(...ys);
		const max = Math.max(...ys);
		const span = max - min || 1;
		const last = Math.max(values.length - 1, 1);
		return points
			.map(([i, v], k) => {
				const x = (i / last) * W;
				const y = H - 4 - ((v - min) / span) * (H - 8);
				return `${k ? 'L' : 'M'}${x.toFixed(1)} ${y.toFixed(1)}`;
			})
			.join(' ');
	});

	const color = $derived(`var(--color-${tone === 'accent' ? 'primary' : tone})`);
</script>

{#if path}
	<svg
		aria-hidden="true"
		class="block h-full w-full"
		viewBox="0 0 {W} {H}"
		preserveAspectRatio="none"
	>
		<path d="{path} L{W} {H} L0 {H} Z" fill={color} opacity="0.12" />
		<path
			d={path}
			fill="none"
			stroke={color}
			stroke-width="1.75"
			stroke-linejoin="round"
			vector-effect="non-scaling-stroke"
		/>
	</svg>
{:else}
	<div class="h-full" aria-hidden="true"></div>
{/if}
