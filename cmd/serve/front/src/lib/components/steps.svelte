<!--
	Something that happens in steps, in order (DESIGN.md §5 Steps): a deployment (queued, started,
	finished) or a job (queued, next try, picked up). Each step has a label, an optional time and a
	note (how long it waited, when it tries again). A dot per step on one line: `accent` when done,
	`danger` when it failed, `faint` when it was stopped on purpose (a cancelled deployment), an outline when still to come, and a spinning arc on the step in progress
	(it pulses instead when motion is reduced). The line fills with `accent` up to the furthest step
	reached. `align="center"` puts each dot over its label in the middle of a narrower band, for a
	page that leads with the steps (the deployment page); `start` keeps them flush left, for drawers.
	A list, so screen readers hear the steps in order and which one is in progress; the line is part
	of the drawing, not a divider.
-->
<script lang="ts">
	import l from '#lib/localization/index.ts';
	import { cn } from '#lib/utils.js';

	export type Step = {
		key: string;
		label: string;
		time?: string;
		note?: string;
		state: 'done' | 'failed' | 'stopped' | 'current' | 'todo';
	};

	type Props = { steps: Step[]; label: string; align?: 'start' | 'center' };

	let { steps, label, align = 'start' }: Props = $props();

	const n = $derived(steps.length);
	const centered = $derived(align === 'center');
	// The furthest step reached: the line is filled up to its dot.
	const reached = $derived(steps.findLastIndex((s) => s.state !== 'todo'));

	// Where the line starts and ends, and how far it fills. Centered columns have no gap and the dots
	// sit in their middle; flush-left columns are 1rem apart and the dots sit 8px from their left edge.
	// Centered steps are larger (owner's call 2026-11-20): 28px dots, a 3px line, 16px labels.
	const half = $derived(centered ? 14 : 8);
	const lineLeft = $derived(centered ? `calc(50% / ${n})` : '8px');
	const lineRight = $derived(
		centered ? `calc(50% / ${n})` : `calc((100% - ${n - 1}rem) / ${n} - ${half}px)`
	);
	const fillWidth = $derived(
		reached <= 0
			? '0px'
			: centered
				? `calc(${reached} * 100% / ${n})`
				: `calc(${reached} * (100% + 1rem) / ${n})`
	);
</script>

<ol
	aria-label={label}
	class={cn('relative grid', centered ? 'mx-auto w-full max-w-220' : 'gap-4')}
	style:grid-template-columns="repeat({n}, minmax(0, 1fr))"
>
	<span
		aria-hidden="true"
		class={cn(
			'absolute bg-line-strong',
			centered ? 'top-[12.5px] h-[3px] rounded-full' : 'top-[7px] h-0.5'
		)}
		style:left={lineLeft}
		style:right={lineRight}
	></span>
	<span
		aria-hidden="true"
		class={cn(
			'absolute bg-primary transition-[width] duration-500 motion-reduce:transition-none',
			centered ? 'top-[12.5px] h-[3px] rounded-full' : 'top-[7px] h-0.5'
		)}
		style:left={lineLeft}
		style:width={fillWidth}
	></span>
	{#each steps as step (step.key)}
		<li
			class={cn(
				'relative flex min-w-0 flex-col pt-6 text-label',
				centered && 'items-center gap-0.5 px-2 pt-10 text-center text-sm'
			)}
			aria-current={step.state === 'current' ? 'step' : undefined}
		>
			<span
				aria-hidden="true"
				class={cn(
					'absolute top-0 z-1 rounded-full bg-surface',
					centered ? 'left-1/2 -ml-3.5 size-7' : 'left-0 size-4',
					centered
						? [
								step.state === 'failed' && 'shadow-[inset_0_0_0_8px_var(--danger)]',
								step.state === 'stopped' && 'shadow-[inset_0_0_0_8px_var(--faint)]',
								step.state === 'done' && 'shadow-[inset_0_0_0_8px_var(--accent)]',
								step.state === 'todo' && 'shadow-[inset_0_0_0_3px_var(--line-strong)]',
								step.state === 'current' &&
									'shadow-[inset_0_0_0_4px_color-mix(in_oklab,var(--accent)_25%,var(--surface))]'
							]
						: [
								step.state === 'failed' && 'shadow-[inset_0_0_0_5px_var(--danger)]',
								step.state === 'stopped' && 'shadow-[inset_0_0_0_5px_var(--faint)]',
								step.state === 'done' && 'shadow-[inset_0_0_0_5px_var(--accent)]',
								step.state === 'todo' && 'shadow-[inset_0_0_0_2px_var(--line-strong)]',
								step.state === 'current' &&
									'shadow-[inset_0_0_0_2px_color-mix(in_oklab,var(--accent)_25%,var(--surface))]'
							]
				)}
			>
				{#if step.state === 'current'}
					<!-- The arc that turns on the faint ring, as thick as the ring. -->
					<span
						class={cn(
							'absolute inset-0 animate-spin rounded-full border-transparent border-t-primary border-r-primary [animation-duration:.8s] motion-reduce:animate-pulse',
							centered ? 'border-4' : 'border-2'
						)}
					></span>
				{/if}
			</span>
			<b
				class={cn(
					'font-semibold',
					centered && 'text-base',
					step.state === 'current' && 'text-primary',
					step.state === 'todo' && 'text-muted-foreground'
				)}
			>
				{step.label}
			</b>
			{#if step.time}<span class="text-muted-foreground tabular-nums">{step.time}</span>{/if}
			{#if step.note}<span class="text-xs text-muted-foreground">{step.note}</span>{/if}
			{#if step.state === 'current'}<span class="sr-only">{l.translate('steps.current')}</span>{/if}
		</li>
	{/each}
</ol>
