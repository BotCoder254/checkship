<!--
	Infrastructure at a glance, as two rows (no inner cards): targets by state and the job queue by
	state, each a link to its page. Bars are proportional; the legend gives the numbers.
-->
<script lang="ts">
	import ArrowRight01Icon from '@hugeicons/core-free-icons/ArrowRight01Icon';
	import ServerStack01Icon from '@hugeicons/core-free-icons/ServerStack01Icon';
	import Tile from '#lib/components/dashboard/tile.svelte';
	import Icon from '#lib/components/icon.svelte';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import { TargetStatus, type Target } from '#lib/resources/targets.ts';

	type Props = {
		targets: Target[] | undefined;
		jobs: { running?: number; pending?: number; failed?: number };
		class?: string;
	};

	let { targets, jobs, class: className }: Props = $props();

	const active = $derived((targets ?? []).filter((t) => !t.cleanup_requested_at));
	const count = (status: TargetStatus) => active.filter((t) => t.state.status === status).length;

	const targetParts = $derived([
		{
			value: count(TargetStatus.Ready),
			color: 'bg-success',
			label: l.translate('dashboard.infra.ready')
		},
		{
			value: count(TargetStatus.Configuring),
			color: 'bg-warning',
			label: l.translate('dashboard.infra.configuring')
		},
		{
			value: count(TargetStatus.Failed),
			color: 'bg-danger',
			label: l.translate('dashboard.infra.failed')
		}
	]);
	const jobParts = $derived([
		{
			value: jobs.running ?? 0,
			color: 'bg-warning',
			label: l.translate('dashboard.infra.running')
		},
		{ value: jobs.pending ?? 0, color: 'bg-faint', label: l.translate('dashboard.infra.pending') },
		{ value: jobs.failed ?? 0, color: 'bg-danger', label: l.translate('dashboard.infra.retrying') }
	]);
</script>

{#snippet row(title: string, href: string, total: number, parts: typeof targetParts)}
	<a
		{href}
		class="grid min-h-11 grid-cols-[minmax(84px,120px)_40px_minmax(0,1fr)_16px] items-center gap-4 rounded-md px-2 py-1.5 transition-colors outline-none hover:bg-hover focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset"
	>
		<span class="text-[13.5px] text-muted-foreground">{title}</span>
		<strong class="text-xl font-semibold tabular-nums">{total}</strong>
		<span class="flex min-w-0 flex-col gap-1.5">
			<span class="flex h-1.5 gap-0.5 overflow-hidden rounded-full bg-raised" aria-hidden="true">
				{#each parts as part (part.label)}
					{#if part.value}<span class={part.color} style="flex: {part.value}"></span>{/if}
				{/each}
			</span>
			<span class="flex flex-wrap gap-x-3.5 gap-y-1 text-[12.5px] text-muted-foreground">
				{#each parts.filter((p) => p.value) as part (part.label)}
					<span class="flex items-center gap-1.5">
						<span class="size-2 rounded-full {part.color}"></span>
						{part.value}
						{part.label}
					</span>
				{:else}
					<span>{l.translate('dashboard.infra.none')}</span>
				{/each}
			</span>
		</span>
		<Icon icon={ArrowRight01Icon} class="size-3.5 text-faint" />
	</a>
{/snippet}

<Tile
	title={l.translate('dashboard.infra')}
	icon={ServerStack01Icon}
	class={className}
	loading={!targets}
>
	<div class="-mx-2 flex flex-col gap-2">
		{@render row(l.translate('breadcrumb.targets'), routes.targets, active.length, targetParts)}
		{@render row(
			l.translate('dashboard.infra.queue'),
			routes.jobs,
			jobParts.reduce((sum, p) => sum + p.value, 0),
			jobParts
		)}
	</div>
</Tile>
