<!--
	Targets: one row per Docker host on a sunken fill, with its configuration state. Rows open the
	target's settings.
-->
<script lang="ts">
	import Add01Icon from '@hugeicons/core-free-icons/Add01Icon';
	import ArrowRight01Icon from '@hugeicons/core-free-icons/ArrowRight01Icon';
	import ServerStack01Icon from '@hugeicons/core-free-icons/ServerStack01Icon';
	import EmptyState from '#lib/components/empty-state.svelte';
	import Icon from '#lib/components/icon.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service, { TargetStatus } from '#lib/resources/targets.ts';
	import { targetBadge } from '#lib/status.ts';

	const { data: targets } = service.queryAll();

	const failing = $derived(
		($targets ?? []).filter((t) => t.state.status === TargetStatus.Failed).length
	);
</script>

<PageHeader title={l.translate('breadcrumb.targets')}>
	{#snippet actions()}
		<Button href={routes.createTarget}>
			<Icon icon={Add01Icon} />
			{l.translate('target.new')}
		</Button>
	{/snippet}
	{#snippet status()}
		{#if $targets}
			<span>{l.translate('targets.summary', [$targets.length, failing])}</span>
		{/if}
	{/snippet}
</PageHeader>

{#if $targets && $targets.length > 0}
	<div role="table" aria-label={l.translate('breadcrumb.targets')} class="flex flex-col gap-1.5">
		<div
			role="row"
			class="hidden grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)_minmax(0,1fr)_20px] gap-4.5 px-4.5 pb-1.5 text-xs font-semibold text-muted-foreground md:grid"
		>
			<span role="columnheader">{l.translate('targets.column.target')}</span>
			<span role="columnheader">{l.translate('targets.column.provider')}</span>
			<span role="columnheader">{l.translate('targets.column.state')}</span>
			<span role="columnheader"
				><span class="sr-only">{l.translate('breadcrumb.targets')}</span></span
			>
		</div>

		{#each $targets as target (target.id)}
			{@const badge = targetBadge(target.state)}
			<a
				role="row"
				href={routes.editTarget(target.id)}
				class="grid grid-cols-1 gap-3 rounded-md bg-sunken px-4 py-3.5 text-ink transition-colors outline-none hover:bg-hover focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset md:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)_minmax(0,1fr)_20px] md:items-center md:gap-4.5 md:px-4.5"
			>
				<span role="cell" class="flex min-w-0 flex-col gap-0.5">
					<span class="truncate text-[15px]/[22px] font-semibold">{target.name}</span>
					<span class="truncate font-mono text-xs text-muted-foreground">
						{target.url ?? l.translate('targets.manual_proxy')}
					</span>
				</span>
				<span role="cell" class="text-sm text-muted-foreground">
					{target.provider.data.host
						? l.translate('targets.remote', [target.provider.data.host])
						: l.translate('targets.local')}
				</span>
				<span role="cell" class="flex flex-col items-start gap-1">
					{#if target.cleanup_requested_at}
						<StatusBadge state="deleting" label={l.translate('apps.deletion_requested')} />
					{:else}
						<StatusBadge state={badge.state} label={badge.label} />
						{#if target.state.error_code}
							<span class="line-clamp-2 text-xs text-muted-foreground">
								{target.state.error_code}
							</span>
						{/if}
					{/if}
				</span>
				<Icon icon={ArrowRight01Icon} class="hidden size-4.5 text-muted-foreground md:block" />
			</a>
		{/each}
	</div>
{:else if $targets}
	<EmptyState
		icon={ServerStack01Icon}
		title={l.translate('targets.empty.title')}
		description={l.translate('targets.empty.description')}
	>
		{#snippet action()}
			<Button href={routes.createTarget}>
				<Icon icon={Add01Icon} />
				{l.translate('target.new')}
			</Button>
		{/snippet}
	</EmptyState>
{/if}
