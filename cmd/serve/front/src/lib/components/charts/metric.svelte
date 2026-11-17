<!--
	One figure (DESIGN.md §11.10 Figures): a small muted label, a large tabular number with its unit,
	and one line comparing it with the previous period in words. The arrow repeats the direction and
	its colour says whether that is good news (green or red), so neither colour nor arrow is the only
	signal. A plain string replaces the comparison when there is nothing to compare.
-->
<script lang="ts">
	import ArrowDownRight01Icon from '@hugeicons/core-free-icons/ArrowDownRight01Icon';
	import ArrowUpRight01Icon from '@hugeicons/core-free-icons/ArrowUpRight01Icon';
	import MinusSignIcon from '@hugeicons/core-free-icons/MinusSignIcon';
	import Icon from '#lib/components/icon.svelte';
	import type { Change } from '#lib/charts.ts';
	import { cn } from '#lib/utils.js';

	type Props = { label: string; value: string; unit?: string; change?: Change | string };

	let { label, value, unit, change }: Props = $props();

	const c = $derived(typeof change === 'string' ? undefined : change);
	const icon = $derived(
		c?.direction === 'up'
			? ArrowUpRight01Icon
			: c?.direction === 'down'
				? ArrowDownRight01Icon
				: c?.direction === 'same'
					? MinusSignIcon
					: undefined
	);
</script>

<div role="group" aria-label={label} class="flex min-w-0 flex-col gap-1">
	<span class="text-label text-muted-foreground">{label}</span>
	<strong class="text-3xl font-semibold tracking-tight tabular-nums"
		>{value}{#if unit}<span class="ml-0.5 text-base font-medium text-muted-foreground">{unit}</span
			>{/if}</strong
	>
	{#if change}
		<span class="flex items-start gap-1.5 text-xs text-muted-foreground">
			{#if icon}
				<Icon
					{icon}
					class={cn(
						'mt-px size-3.5 shrink-0',
						c?.good === true && 'text-success',
						c?.good === false && 'text-danger'
					)}
				/>
			{/if}
			{typeof change === 'string' ? change : change.text}
		</span>
	{/if}
</div>
