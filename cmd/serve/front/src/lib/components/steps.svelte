<!--
	Something that happens in steps, in order (DESIGN.md §5 Steps): a deployment (queued, started,
	finished) or a job (queued, next try, picked up). Each step has a label, an optional time and a
	note (how long it waited, when it tries again). A dot per step on one line: `accent` when done,
	`danger` when it failed, an outline when still to come (pulsing when it is the current one). A
	list, so screen readers hear the steps in order; the line is part of the drawing, not a divider.
-->
<script lang="ts">
	import { cn } from '#lib/utils.js';

	export type Step = {
		key: string;
		label: string;
		time?: string;
		note?: string;
		state: 'done' | 'failed' | 'current' | 'todo';
	};

	type Props = { steps: Step[]; label: string };

	let { steps, label }: Props = $props();
</script>

<ol
	aria-label={label}
	class="relative grid gap-4"
	style:grid-template-columns="repeat({steps.length}, minmax(0, 1fr))"
>
	<span
		aria-hidden="true"
		class="absolute top-[7px] left-2 h-0.5 bg-line-strong"
		style:right="calc({100 / steps.length}% - 8px)"
	></span>
	{#each steps as step (step.key)}
		<li class="relative flex min-w-0 flex-col pt-6 text-label">
			<span
				aria-hidden="true"
				class={cn(
					'absolute top-0 left-0 size-4 rounded-full bg-surface',
					step.state === 'failed' && 'shadow-[inset_0_0_0_5px_var(--danger)]',
					step.state === 'done' && 'shadow-[inset_0_0_0_5px_var(--accent)]',
					(step.state === 'todo' || step.state === 'current') &&
						'shadow-[inset_0_0_0_2px_var(--line-strong)]',
					step.state === 'current' && 'animate-pulse motion-reduce:animate-none'
				)}
			></span>
			<b class="font-semibold">{step.label}</b>
			{#if step.time}<span class="text-muted-foreground tabular-nums">{step.time}</span>{/if}
			{#if step.note}<span class="text-xs text-muted-foreground">{step.note}</span>{/if}
		</li>
	{/each}
</ol>
