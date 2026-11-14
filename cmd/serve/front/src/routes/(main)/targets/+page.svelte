<!--
	Targets (DESIGN.md §11.8): one row per Docker engine with where it is, how it exposes services,
	its state (Ready shows when it was configured, Failed its error code) and how many applications
	deploy to it. A failed row offers Reconfigure right there. Rows open the target's overview.
-->
<script lang="ts">
	import Add01Icon from '@hugeicons/core-free-icons/Add01Icon';
	import ArrowRight01Icon from '@hugeicons/core-free-icons/ArrowRight01Icon';
	import Globe02Icon from '@hugeicons/core-free-icons/Globe02Icon';
	import Loading03Icon from '@hugeicons/core-free-icons/Loading03Icon';
	import RefreshIcon from '@hugeicons/core-free-icons/RefreshIcon';
	import ServerStack01Icon from '@hugeicons/core-free-icons/ServerStack01Icon';
	import { toast } from 'svelte-sonner';
	import EmptyState from '#lib/components/empty-state.svelte';
	import Icon from '#lib/components/icon.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import appsService from '#lib/resources/apps.ts';
	import service, { TargetStatus, type Target } from '#lib/resources/targets.ts';
	import { targetBadge } from '#lib/status.ts';
	import { connectionLabel, isLocal, usesOf } from '#lib/target.ts';
	import { cn } from '#lib/utils.js';

	const { data: targets } = service.queryAll();
	const { data: apps } = appsService.queryAll();

	const count = (status: TargetStatus) =>
		($targets ?? []).filter((t) => !t.cleanup_requested_at && t.state.status === status).length;
	const deleting = $derived(($targets ?? []).filter((t) => t.cleanup_requested_at).length);

	let reconfiguring = $state<string>();

	async function reconfigure(target: Target) {
		reconfiguring = target.id;
		try {
			await service.reconfigure(target.id);
			await service.fetchAll(undefined, { cache: 'no-store' });
		} catch (ex) {
			toast.error(ex instanceof Error ? ex.message : l.translate('unexpected_error'));
		} finally {
			reconfiguring = undefined;
		}
	}

	const columns = 'md:grid-cols-[36px_minmax(0,1.25fr)_minmax(0,1fr)_190px_120px_132px_16px]';
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
			<span>{l.translate('targets.count', [$targets.length])}</span>
			{#if count(TargetStatus.Ready)}
				<span class="flex items-center gap-1.5">
					<span class="size-2 rounded-full bg-success"></span>
					{l.translate('targets.count.ready', [count(TargetStatus.Ready)])}
				</span>
			{/if}
			{#if count(TargetStatus.Configuring)}
				<span class="flex items-center gap-1.5">
					<Icon icon={Loading03Icon} class="size-3.5 animate-spin text-warning" />
					{l.translate('targets.count.configuring', [count(TargetStatus.Configuring)])}
				</span>
			{/if}
			{#if count(TargetStatus.Failed)}
				<span class="flex items-center gap-1.5">
					<span class="size-2 rounded-full bg-danger"></span>
					{l.translate('targets.count.failed', [count(TargetStatus.Failed)])}
				</span>
			{/if}
			{#if deleting}
				<span>{l.translate('targets.count.deleting', [deleting])}</span>
			{/if}
		{/if}
	{/snippet}
</PageHeader>

{#if $targets && $targets.length > 0}
	<div role="table" aria-label={l.translate('breadcrumb.targets')} class="flex flex-col gap-1.5">
		<div
			role="row"
			class={cn(
				'hidden gap-4 px-3.5 pb-0.5 text-xs font-semibold text-muted-foreground md:grid',
				columns
			)}
		>
			<span role="columnheader"></span>
			<span role="columnheader">{l.translate('targets.column.target')}</span>
			<span role="columnheader">{l.translate('targets.column.exposure')}</span>
			<span role="columnheader">{l.translate('targets.column.state')}</span>
			<span role="columnheader">{l.translate('targets.column.used_by')}</span>
			<span role="columnheader"
				><span class="sr-only">{l.translate('detail.row.actions')}</span></span
			>
			<span role="columnheader"></span>
		</div>

		{#each $targets as target (target.id)}
			{@const badge = targetBadge(target.state)}
			{@const failed = !target.cleanup_requested_at && target.state.status === TargetStatus.Failed}
			{@const uses = $apps ? usesOf($apps, target.id) : undefined}
			{@const appsUsing = uses ? new Set(uses.map((u) => u.app.id)).size : undefined}
			<div
				role="row"
				class={cn(
					'relative grid grid-cols-[36px_minmax(0,1fr)] items-center gap-x-4 gap-y-2 rounded-md bg-sunken px-3.5 py-3 text-ink transition-colors hover:bg-hover md:min-h-16 md:py-2.5',
					columns,
					target.cleanup_requested_at && 'opacity-70'
				)}
			>
				<span
					role="cell"
					class={cn(
						'flex size-9 items-center justify-center rounded-lg max-md:row-span-2',
						failed ? 'bg-danger-soft text-danger' : 'bg-raised text-muted-foreground'
					)}
				>
					<Icon icon={isLocal(target) ? ServerStack01Icon : Globe02Icon} class="size-4.5" />
				</span>
				<span role="cell" class="flex min-w-0 flex-col">
					<!-- The whole row opens the target; the link stretches over it. -->
					<a
						href={routes.target(target.id)}
						class="truncate text-[15px]/[22px] font-semibold outline-none after:absolute after:inset-0 after:rounded-md focus-visible:after:ring-2 focus-visible:after:ring-ring focus-visible:after:ring-inset"
					>
						{target.name}
					</a>
					<span
						class={cn(
							'truncate text-[12.5px] text-muted-foreground',
							!isLocal(target) && 'font-mono'
						)}
					>
						{connectionLabel(target)}
					</span>
				</span>
				<span role="cell" class="flex min-w-0 flex-col max-md:col-start-2">
					{#if target.url}
						<span class="truncate font-mono text-[12.5px] text-primary">{target.url}</span>
						<span class="text-xs text-faint">{l.translate('targets.exposure.automatic')}</span>
					{:else}
						<span class="text-[13px] text-muted-foreground"
							>{l.translate('targets.manual_proxy')}</span
						>
						<span class="text-xs text-faint">{l.translate('targets.exposure.manual')}</span>
					{/if}
				</span>
				<span role="cell" class="flex flex-col items-start gap-1 max-md:col-start-2">
					{#if target.cleanup_requested_at}
						<StatusBadge state="deleting" label={l.translate('targets.state.deleting')} />
					{:else}
						<StatusBadge state={badge.state} label={badge.label} />
						{#if failed && target.state.error_code}
							<span
								title={target.state.error_code}
								class="max-w-full truncate font-mono text-[11.5px] text-danger"
							>
								{target.state.error_code}
							</span>
						{:else if target.state.status === TargetStatus.Ready && target.state.last_ready_version}
							<span class="text-[11.5px] text-faint">
								{l.translate('targets.configured_at', [
									l.datetime(target.state.last_ready_version)
								])}
							</span>
						{/if}
					{/if}
				</span>
				<span role="cell" class="truncate max-md:col-start-2">
					<span class="text-[13px] text-muted-foreground">
						{appsUsing === undefined
							? ''
							: appsUsing === 0
								? l.translate('targets.used_by.none')
								: l.translate('targets.used_by', [appsUsing])}
					</span>
				</span>
				<span role="cell" class="flex justify-end max-md:col-start-2 max-md:justify-start">
					{#if failed}
						<Button
							variant="outline"
							size="sm"
							class="relative z-10"
							loading={reconfiguring === target.id}
							onclick={() => reconfigure(target)}
						>
							<Icon icon={RefreshIcon} />
							{l.translate('target.reconfigure')}
						</Button>
					{/if}
				</span>
				<Icon icon={ArrowRight01Icon} class="hidden size-4 text-faint md:block" />
			</div>
		{/each}
		<p class="mt-2 text-[13px] text-muted-foreground">{l.translate('targets.explainer')}</p>
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
