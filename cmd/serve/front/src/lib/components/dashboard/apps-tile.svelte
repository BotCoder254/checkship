<!--
	Applications over the period: deploys, a success bar, failures and the last deploy, sorted by
	the user's choice and paged (offset pagination: the list is small and page numbers help).
-->
<script lang="ts">
	import ArrowLeft01Icon from '@hugeicons/core-free-icons/ArrowLeft01Icon';
	import ArrowRight01Icon from '@hugeicons/core-free-icons/ArrowRight01Icon';
	import Layers01Icon from '@hugeicons/core-free-icons/Layers01Icon';
	import { untrack } from 'svelte';
	import Tile from '#lib/components/dashboard/tile.svelte';
	import Icon from '#lib/components/icon.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service, { type AppsSort, type DashboardFilters } from '#lib/resources/dashboard.ts';
	import { cn } from '#lib/utils.js';

	type Props = { filters: DashboardFilters; refresh: number; class?: string };

	let { filters, refresh, class: className }: Props = $props();

	const PER_PAGE = 6;

	let sort = $state<AppsSort>('deploys');
	let page = $state(1);

	const params = $derived({ ...filters, sort, page, perPage: PER_PAGE });
	const query = $derived(service.queryApps(params));
	const data = $derived(query.data);
	const error = $derived(query.error);
	const apps = $derived($data);

	// Filters changed: back to the first page.
	$effect(() => {
		void filters.range;
		void filters.environment;
		untrack(() => (page = 1));
	});

	$effect(() => {
		if (refresh === 0) return;
		untrack(() => service.fetchApps(params, { cache: 'no-store' }).catch(() => {}));
	});

	const sorts: { value: AppsSort; label: string }[] = [
		{ value: 'deploys', label: l.translate('dashboard.apps.sort.deploys') },
		{ value: 'failures', label: l.translate('dashboard.apps.sort.failures') },
		{ value: 'recent', label: l.translate('dashboard.apps.sort.recent') }
	];

	const pages = $derived(apps ? Math.max(1, Math.ceil(apps.total / apps.per_page)) : 1);
	const max = $derived(Math.max(1, ...(apps?.data ?? []).map((a) => a.deployments)));
	const columns =
		'grid-cols-[minmax(0,1fr)_52px_52px] sm:grid-cols-[minmax(0,1fr)_52px_minmax(80px,130px)_52px_100px]';
</script>

<Tile
	title={l.translate('breadcrumb.applications')}
	icon={Layers01Icon}
	class={className}
	loading={!apps && !$error}
	error={!apps ? $error : undefined}
	onretry={() => service.fetchApps(params, { cache: 'no-store' }).catch(() => {})}
>
	{#snippet actions()}
		<Select.Root
			type="single"
			value={sort}
			onValueChange={(value) => {
				sort = value as AppsSort;
				page = 1;
			}}
		>
			<Select.Trigger size="sm" class="w-40" aria-label={l.translate('dashboard.apps.sort')}>
				{sorts.find((s) => s.value === sort)?.label}
			</Select.Trigger>
			<Select.Content>
				{#each sorts as s (s.value)}
					<Select.Item value={s.value} label={s.label}>{s.label}</Select.Item>
				{/each}
			</Select.Content>
		</Select.Root>
	{/snippet}

	{#if apps && apps.data.length > 0}
		<div
			role="table"
			aria-label={l.translate('breadcrumb.applications')}
			class="flex flex-col gap-1.5"
		>
			<div
				role="row"
				class={cn('grid gap-3 px-3 text-xs font-semibold text-muted-foreground', columns)}
			>
				<span role="columnheader">{l.translate('dashboard.apps.application')}</span>
				<span role="columnheader" class="text-right">{l.translate('dashboard.apps.deploys')}</span>
				<span role="columnheader" class="max-sm:hidden"
					>{l.translate('dashboard.apps.success')}</span
				>
				<span role="columnheader" class="text-right">{l.translate('dashboard.series.failed')}</span>
				<span role="columnheader" class="text-right max-sm:hidden"
					>{l.translate('dashboard.apps.last')}</span
				>
			</div>
			{#each apps.data as app (app.id)}
				<div
					role="row"
					class={cn(
						'grid min-h-11 items-center gap-3 rounded-md bg-surface px-3 text-[13px]',
						columns
					)}
				>
					<a
						role="cell"
						href={routes.app(app.id)}
						class="truncate text-sm font-medium hover:underline"
					>
						{app.name}
					</a>
					<span role="cell" class="text-right tabular-nums">{app.deployments}</span>
					<span role="cell" class="flex flex-col gap-0.5 max-sm:hidden">
						{#if app.deployments > 0}
							<span
								class="flex h-1.5 gap-px overflow-hidden rounded-full"
								style="width: {Math.max(18, (app.deployments / max) * 100)}%"
								aria-hidden="true"
							>
								<span class="bg-success" style="flex: {app.succeeded}"></span>
								{#if app.failed}<span class="bg-danger" style="flex: {app.failed}"></span>{/if}
								{#if app.deployments - app.succeeded - app.failed > 0}
									<span
										class="bg-faint"
										style="flex: {app.deployments - app.succeeded - app.failed}"
									></span>
								{/if}
							</span>
							<span class="text-[11.5px] text-faint">
								{app.success_rate == null
									? '—'
									: l.translate('dashboard.apps.ok', [Math.round(app.success_rate * 100)])}
							</span>
						{:else}
							<span class="text-[11.5px] text-faint">—</span>
						{/if}
					</span>
					<span
						role="cell"
						class={cn('text-right tabular-nums', app.failed ? 'text-danger' : 'text-faint')}
					>
						{app.failed}
					</span>
					<span role="cell" class="truncate text-right text-muted-foreground max-sm:hidden">
						{app.last_deployment ? l.relative(app.last_deployment) : '—'}
					</span>
				</div>
			{/each}
		</div>
		<div class="mt-auto flex items-center justify-between gap-3 pt-1 text-[12.5px] text-faint">
			<span>{l.translate('dashboard.apps.page', [apps.total, apps.page, pages])}</span>
			{#if !(apps.first_page && apps.last_page)}
				<span class="flex gap-1">
					<Button
						variant="outline"
						size="icon-sm"
						aria-label={l.translate('pagination.previous')}
						disabled={apps.first_page}
						onclick={() => page--}
					>
						<Icon icon={ArrowLeft01Icon} />
					</Button>
					<Button
						variant="outline"
						size="icon-sm"
						aria-label={l.translate('pagination.next')}
						disabled={apps.last_page}
						onclick={() => page++}
					>
						<Icon icon={ArrowRight01Icon} />
					</Button>
				</span>
			{/if}
		</div>
	{:else if apps}
		<p class="flex flex-1 items-center justify-center py-8 text-[13px] text-muted-foreground">
			{l.translate('dashboard.apps.empty')}
		</p>
	{/if}
</Tile>
