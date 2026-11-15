<!--
	Applications over the period: deploys, a success bar, failures and the last deploy, sorted by
	the user's choice and paged (offset pagination: the list is small and page numbers help).
-->
<script lang="ts">
	import ArrowLeft01Icon from '@hugeicons/core-free-icons/ArrowLeft01Icon';
	import ArrowRight01Icon from '@hugeicons/core-free-icons/ArrowRight01Icon';
	import CancelCircleIcon from '@hugeicons/core-free-icons/CancelCircleIcon';
	import CheckmarkCircle02Icon from '@hugeicons/core-free-icons/CheckmarkCircle02Icon';
	import Clock01Icon from '@hugeicons/core-free-icons/Clock01Icon';
	import CommandLineIcon from '@hugeicons/core-free-icons/CommandLineIcon';
	import Layers01Icon from '@hugeicons/core-free-icons/Layers01Icon';
	import Rocket01Icon from '@hugeicons/core-free-icons/Rocket01Icon';
	import SortingDownIcon from '@hugeicons/core-free-icons/SortingDownIcon';
	import TextIcon from '@hugeicons/core-free-icons/TextIcon';
	import ViewIcon from '@hugeicons/core-free-icons/ViewIcon';
	import { untrack } from 'svelte';
	import Tile from '#lib/components/dashboard/tile.svelte';
	import Icon from '#lib/components/icon.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import FilterMenu, { type FilterOption } from '#lib/components/table/filter-menu.svelte';
	import RowMenuGroup from '#lib/components/table/row-menu-group.svelte';
	import RowMenuItem from '#lib/components/table/row-menu-item.svelte';
	import RowMenu from '#lib/components/table/row-menu.svelte';
	import * as Table from '#lib/components/ui/table/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service, { type AppsSort, type DashboardFilters } from '#lib/resources/dashboard.ts';
	import { ACTIONS_CELL, ACTIONS_HEAD, TableColumns } from '#lib/table-columns.svelte.ts';

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

	const sorts: FilterOption<AppsSort>[] = [
		{ value: 'deploys', label: l.translate('dashboard.apps.sort.deploys'), icon: Rocket01Icon },
		{
			value: 'failures',
			label: l.translate('dashboard.apps.sort.failures'),
			icon: CancelCircleIcon
		},
		{ value: 'recent', label: l.translate('dashboard.apps.sort.recent'), icon: Clock01Icon }
	];

	const columns = new TableColumns([
		{ id: 'name', label: l.translate('column.name'), icon: TextIcon, width: 'w-[30%]' },
		{
			id: 'deploys',
			label: l.translate('column.deploys'),
			icon: Rocket01Icon,
			width: 'w-[16%]',
			align: 'right'
		},
		{
			id: 'success',
			label: l.translate('column.success'),
			icon: CheckmarkCircle02Icon,
			width: 'w-[22%]',
			priority: 2
		},
		{
			id: 'failed',
			label: l.translate('column.failed'),
			icon: CancelCircleIcon,
			width: 'w-[14%]',
			align: 'right'
		},
		{
			id: 'last',
			label: l.translate('column.last'),
			icon: Clock01Icon,
			width: 'w-[18%]',
			priority: 3
		}
	]);

	const pages = $derived(apps ? Math.max(1, Math.ceil(apps.total / apps.per_page)) : 1);
	const max = $derived(Math.max(1, ...(apps?.data ?? []).map((a) => a.deployments)));
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
		<FilterMenu
			label={l.translate('dashboard.apps.sort')}
			icon={SortingDownIcon}
			value={sort}
			options={sorts}
			onValueChange={(value) => {
				sort = value;
				page = 1;
			}}
		/>
	{/snippet}

	{#if apps && apps.data.length > 0}
		<Table.Root aria-label={l.translate('breadcrumb.applications')}>
			<Table.Header>
				<Table.Row>
					{#each columns.shown as column (column.id)}
						<Table.Head class={columns.head(column.id)}>{column.label}</Table.Head>
					{/each}
					<Table.Head class={ACTIONS_HEAD}>{l.translate('column.actions')}</Table.Head>
				</Table.Row>
			</Table.Header>
			<Table.Body>
				{#each apps.data as app (app.id)}
					<Table.Row class="relative">
						<Table.Cell class={columns.cell('name')}>
							<a
								href={routes.app(app.id)}
								title={app.name}
								class="text-sm font-semibold outline-none after:absolute after:inset-0 after:rounded-md focus-visible:after:ring-2 focus-visible:after:ring-ring focus-visible:after:ring-inset"
								>{app.name}</a
							>
						</Table.Cell>
						<Table.Cell class={columns.cell('deploys', 'tabular-nums')}
							>{app.deployments}</Table.Cell
						>
						<Table.Cell class={columns.cell('success')}>
							{#if app.deployments > 0}
								<span class="flex items-center gap-2">
									<span
										class="flex h-1.5 min-w-0 flex-1 gap-px overflow-hidden rounded-full"
										style="max-width: {Math.max(18, (app.deployments / max) * 100)}%"
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
									<span class="shrink-0 text-[11.5px] text-faint tabular-nums">
										{app.success_rate == null ? '—' : `${Math.round(app.success_rate * 100)}%`}
									</span>
								</span>
							{:else}
								<span class="text-faint">—</span>
							{/if}
						</Table.Cell>
						<Table.Cell
							class={columns.cell(
								'failed',
								app.failed ? 'text-danger tabular-nums' : 'text-faint tabular-nums'
							)}
						>
							{app.failed}
						</Table.Cell>
						<Table.Cell class={columns.cell('last', 'text-muted-foreground')}>
							{app.last_deployment ? l.relative(app.last_deployment) : '—'}
						</Table.Cell>
						<Table.Cell class={ACTIONS_CELL}>
							<RowMenu label={l.translate('table.menu', [app.name])}>
								<RowMenuGroup>
									<RowMenuItem icon={ViewIcon} href={routes.app(app.id)}>
										{l.translate('menu.open')}
									</RowMenuItem>
									<RowMenuItem icon={Rocket01Icon} href={routes.createDeployment(app.id)}>
										{l.translate('menu.deploy')}
									</RowMenuItem>
									<RowMenuItem icon={CommandLineIcon} href={routes.appLogs(app.id)}>
										{l.translate('menu.logs')}
									</RowMenuItem>
								</RowMenuGroup>
							</RowMenu>
						</Table.Cell>
					</Table.Row>
				{/each}
			</Table.Body>
		</Table.Root>
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
