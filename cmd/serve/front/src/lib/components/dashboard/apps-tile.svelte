<!--
	Applications over the period, as a list (DESIGN.md §5 Lists): the success bar beside the name;
	deploys, failures and the last deploy on the details line; sorted by the user's choice and paged.
-->
<script lang="ts">
	import ArrowLeft01Icon from '@hugeicons/core-free-icons/ArrowLeft01Icon';
	import ArrowRight01Icon from '@hugeicons/core-free-icons/ArrowRight01Icon';
	import CancelCircleIcon from '@hugeicons/core-free-icons/CancelCircleIcon';
	import Clock01Icon from '@hugeicons/core-free-icons/Clock01Icon';
	import CommandLineIcon from '@hugeicons/core-free-icons/CommandLineIcon';
	import DashboardSquare01Icon from '@hugeicons/core-free-icons/DashboardSquare01Icon';
	import Layers01Icon from '@hugeicons/core-free-icons/Layers01Icon';
	import Rocket01Icon from '@hugeicons/core-free-icons/Rocket01Icon';
	import SortingDownIcon from '@hugeicons/core-free-icons/SortingDownIcon';
	import ViewIcon from '@hugeicons/core-free-icons/ViewIcon';
	import { untrack } from 'svelte';
	import Tile from '#lib/components/dashboard/tile.svelte';
	import Icon from '#lib/components/icon.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import FilterMenu, { type FilterOption } from '#lib/components/table/filter-menu.svelte';
	import RowMenuGroup from '#lib/components/table/row-menu-group.svelte';
	import RowMenuItem from '#lib/components/table/row-menu-item.svelte';
	import RowMenu from '#lib/components/table/row-menu.svelte';
	import TimelineList from '#lib/components/timeline/timeline-list.svelte';
	import TimelineRow from '#lib/components/timeline/timeline-row.svelte';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service, { type AppsSort, type DashboardFilters } from '#lib/resources/dashboard.ts';

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

	const pages = $derived(apps ? Math.max(1, Math.ceil(apps.total / apps.per_page)) : 1);
	type AppStat = NonNullable<typeof apps>['data'][number];

	function detailsOf(app: AppStat): string[] {
		return [
			l.translate('dashboard.apps.row.deploys', [app.deployments]),
			app.failed ? l.translate('dashboard.apps.row.failed', [app.failed]) : undefined,
			app.last_deployment
				? l.translate('dashboard.apps.row.last', [l.relative(app.last_deployment)])
				: undefined
		].filter((part): part is string => !!part);
	}

	const rate = (app: AppStat) =>
		app.success_rate == null ? undefined : `${Math.round(app.success_rate * 100)}%`;

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
		<TimelineList label={l.translate('breadcrumb.applications')}>
			{#each apps.data as app (app.id)}
				<TimelineRow
					icon={DashboardSquare01Icon}
					tone={app.failed ? 'danger' : app.deployments ? 'success' : 'muted'}
					label={[
						app.name,
						...detailsOf(app),
						rate(app) && l.translate('dashboard.apps.row.success', [rate(app)!])
					]
						.filter(Boolean)
						.join(', ')}
					href={routes.app(app.id)}
				>
					{#snippet title()}{app.name}{/snippet}
					{#snippet badge()}
						{#if app.deployments > 0}
							<span class="flex w-24 shrink-0 items-center gap-2 font-normal">
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
								<span class="shrink-0 text-xs text-muted-foreground tabular-nums">
									{rate(app) ?? '—'}
								</span>
							</span>
						{/if}
					{/snippet}
					{#snippet details()}{detailsOf(app).join(' · ')}{/snippet}
					{#snippet actions()}
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
					{/snippet}
				</TimelineRow>
			{/each}
		</TimelineList>
		<div class="mt-auto flex items-center justify-between gap-3 pt-1 text-xs text-muted-foreground">
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
		<p class="flex flex-1 items-center justify-center py-8 text-label text-muted-foreground">
			{l.translate('dashboard.apps.empty')}
		</p>
	{/if}
</Tile>
