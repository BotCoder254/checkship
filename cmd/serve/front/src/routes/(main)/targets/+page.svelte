<!--
	Targets (DESIGN.md §5 Tables, §11.8): one table row per Docker engine with where it is, the URL it
	exposes services on, its state (Failed shows the error code where Configured goes) and how many
	applications deploy to it. The name opens the target; the ⋯ menu has Open, Reconfigure (failed
	targets), Settings and Delete (only while no application uses it).
-->
<script lang="ts">
	import Add01Icon from '@hugeicons/core-free-icons/Add01Icon';
	import Calendar03Icon from '@hugeicons/core-free-icons/Calendar03Icon';
	import CheckmarkCircle02Icon from '@hugeicons/core-free-icons/CheckmarkCircle02Icon';
	import CancelCircleIcon from '@hugeicons/core-free-icons/CancelCircleIcon';
	import DashboardSquare01Icon from '@hugeicons/core-free-icons/DashboardSquare01Icon';
	import Delete02Icon from '@hugeicons/core-free-icons/Delete02Icon';
	import Globe02Icon from '@hugeicons/core-free-icons/Globe02Icon';
	import Layers01Icon from '@hugeicons/core-free-icons/Layers01Icon';
	import Loading03Icon from '@hugeicons/core-free-icons/Loading03Icon';
	import RefreshIcon from '@hugeicons/core-free-icons/RefreshIcon';
	import Search01Icon from '@hugeicons/core-free-icons/Search01Icon';
	import ServerStack01Icon from '@hugeicons/core-free-icons/ServerStack01Icon';
	import Settings02Icon from '@hugeicons/core-free-icons/Settings02Icon';
	import TextIcon from '@hugeicons/core-free-icons/TextIcon';
	import ViewIcon from '@hugeicons/core-free-icons/ViewIcon';
	import { toast } from 'svelte-sonner';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import EmptyState from '#lib/components/empty-state.svelte';
	import Icon from '#lib/components/icon.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import ColumnsMenu from '#lib/components/table/columns-menu.svelte';
	import FilterMenu, { type FilterOption } from '#lib/components/table/filter-menu.svelte';
	import RowMenuGroup from '#lib/components/table/row-menu-group.svelte';
	import RowMenuItem from '#lib/components/table/row-menu-item.svelte';
	import RowMenu from '#lib/components/table/row-menu.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import appsService from '#lib/resources/apps.ts';
	import service, { TargetStatus, type Target } from '#lib/resources/targets.ts';
	import { targetBadge } from '#lib/status.ts';
	import { ACTIONS_CELL, ACTIONS_HEAD, TableColumns } from '#lib/table-columns.svelte.ts';
	import { connectionLabel, isLocal, usesOf } from '#lib/target.ts';

	const { data: targets } = service.queryAll();
	const { data: apps } = appsService.queryAll();

	const count = (status: TargetStatus) =>
		($targets ?? []).filter((t) => !t.cleanup_requested_at && t.state.status === status).length;
	const deleting = $derived(($targets ?? []).filter((t) => t.cleanup_requested_at).length);

	const columns = new TableColumns(
		[
			{
				id: 'name',
				label: l.translate('column.name'),
				icon: TextIcon,
				width: 'w-[16%]',
				locked: true
			},
			{
				id: 'host',
				label: l.translate('column.host'),
				icon: ServerStack01Icon,
				width: 'w-[19%]',
				priority: 3
			},
			{
				id: 'url',
				label: l.translate('column.url'),
				icon: Globe02Icon,
				width: 'w-[21%]',
				priority: 2
			},
			{
				id: 'state',
				label: l.translate('column.state'),
				icon: CheckmarkCircle02Icon,
				width: 'w-[14%]',
				locked: true
			},
			{
				id: 'configured',
				label: l.translate('column.configured'),
				icon: Calendar03Icon,
				width: 'w-[16%]',
				priority: 3
			},
			{
				id: 'apps',
				label: l.translate('column.apps'),
				icon: DashboardSquare01Icon,
				width: 'w-[9%]',
				priority: 2
			}
		],
		'targets'
	);

	type StateFilter = 'any' | 'ready' | 'configuring' | 'failed';
	let search = $state('');
	let stateFilter = $state<StateFilter>('any');
	const stateOptions: FilterOption<StateFilter>[] = [
		{ value: 'any', label: l.translate('detail.filter.any'), icon: Layers01Icon },
		{
			value: 'ready',
			label: l.translate('status.ready'),
			icon: CheckmarkCircle02Icon,
			tone: 'text-success'
		},
		{
			value: 'configuring',
			label: l.translate('status.configuring'),
			icon: Loading03Icon,
			tone: 'text-warning'
		},
		{
			value: 'failed',
			label: l.translate('status.failed'),
			icon: CancelCircleIcon,
			tone: 'text-danger'
		}
	];
	const stateValue: Record<StateFilter, TargetStatus | undefined> = {
		any: undefined,
		ready: TargetStatus.Ready,
		configuring: TargetStatus.Configuring,
		failed: TargetStatus.Failed
	};

	const shown = $derived(
		($targets ?? []).filter(
			(t) =>
				(stateFilter === 'any' ||
					(!t.cleanup_requested_at && t.state.status === stateValue[stateFilter])) &&
				t.name.toLowerCase().includes(search.trim().toLowerCase())
		)
	);

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

	let removing = $state<Target>();
	let deleteOpen = $state(false);

	async function remove() {
		await service.delete(removing!.id);
		await service.fetchAll(undefined, { cache: 'no-store' });
	}
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
	<div class="flex flex-col gap-3">
		<div class="flex flex-wrap items-center justify-between gap-2">
			<div class="flex flex-wrap items-center gap-2">
				<div class="relative w-72 max-w-full">
					<Input
						type="search"
						bind:value={search}
						placeholder={l.translate('targets.filter')}
						aria-label={l.translate('targets.filter')}
						class="pr-9"
					/>
					<Icon
						icon={Search01Icon}
						class="pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2 text-muted-foreground"
					/>
				</div>
				<FilterMenu
					label={l.translate('column.state')}
					icon={CheckmarkCircle02Icon}
					bind:value={stateFilter}
					options={stateOptions}
				/>
			</div>
			<ColumnsMenu {columns} />
		</div>

		<Table.Root aria-label={l.translate('breadcrumb.targets')}>
			<Table.Header>
				<Table.Row>
					{#each columns.shown as column (column.id)}
						<Table.Head class={columns.head(column.id)}>{column.label}</Table.Head>
					{/each}
					<Table.Head class={ACTIONS_HEAD}>{l.translate('column.actions')}</Table.Head>
				</Table.Row>
			</Table.Header>
			<Table.Body>
				{#each shown as target (target.id)}
					{@const badge = targetBadge(target.state)}
					{@const failed =
						!target.cleanup_requested_at && target.state.status === TargetStatus.Failed}
					{@const uses = $apps ? usesOf($apps, target.id) : undefined}
					{@const appsUsing = uses ? new Set(uses.map((u) => u.app.id)).size : undefined}
					<Table.Row class="relative">
						{#if columns.visible('name')}
							<Table.Cell class={columns.cell('name')}>
								<!-- The whole row opens the target; the link stretches over it. -->
								<a
									href={routes.target(target.id)}
									title={target.name}
									class="text-sm font-semibold outline-none after:absolute after:inset-0 after:rounded-md focus-visible:after:ring-2 focus-visible:after:ring-ring focus-visible:after:ring-inset"
									>{target.name}</a
								>
							</Table.Cell>
						{/if}
						{#if columns.visible('host')}
							<Table.Cell
								class={columns.cell(
									'host',
									isLocal(target)
										? 'text-muted-foreground'
										: 'font-mono text-[12.5px] text-muted-foreground'
								)}
								title={connectionLabel(target)}
							>
								{connectionLabel(target)}
							</Table.Cell>
						{/if}
						{#if columns.visible('url')}
							<Table.Cell
								class={columns.cell(
									'url',
									target.url ? 'font-mono text-[12.5px] text-primary' : 'text-muted-foreground'
								)}
								title={target.url ?? l.translate('targets.manual_proxy')}
							>
								{target.url ?? l.translate('targets.url.manual')}
							</Table.Cell>
						{/if}
						{#if columns.visible('state')}
							<Table.Cell class={columns.cell('state')}>
								{#if target.cleanup_requested_at}
									<StatusBadge state="deleting" label={l.translate('targets.state.deleting')} />
								{:else}
									<StatusBadge state={badge.state} label={badge.label} />
								{/if}
							</Table.Cell>
						{/if}
						{#if columns.visible('configured')}
							{#if failed && target.state.error_code}
								<Table.Cell
									class={columns.cell('configured', 'font-mono text-[12.5px] text-danger')}
									title={target.state.error_code}
								>
									{target.state.error_code}
								</Table.Cell>
							{:else}
								<Table.Cell class={columns.cell('configured', 'text-muted-foreground')}>
									{target.state.last_ready_version
										? l.datetime(target.state.last_ready_version)
										: '—'}
								</Table.Cell>
							{/if}
						{/if}
						{#if columns.visible('apps')}
							<Table.Cell class={columns.cell('apps', 'text-muted-foreground tabular-nums')}>
								{appsUsing ?? ''}
							</Table.Cell>
						{/if}
						<Table.Cell class={ACTIONS_CELL}>
							<RowMenu label={l.translate('table.menu', [target.name])}>
								<RowMenuGroup>
									<RowMenuItem icon={ViewIcon} href={routes.target(target.id)}>
										{l.translate('menu.open')}
									</RowMenuItem>
									{#if failed}
										<RowMenuItem
											icon={RefreshIcon}
											disabled={reconfiguring === target.id}
											onSelect={() => reconfigure(target)}
										>
											{l.translate('target.reconfigure')}
										</RowMenuItem>
									{/if}
									{#if !target.cleanup_requested_at}
										<RowMenuItem icon={Settings02Icon} href={routes.editTarget(target.id)}>
											{l.translate('menu.settings')}
										</RowMenuItem>
									{/if}
								</RowMenuGroup>
								{#if !target.cleanup_requested_at && target.state.status !== TargetStatus.Configuring}
									<RowMenuGroup>
										{#if appsUsing}
											<!-- In use: the settings page lists what deploys to it. -->
											<RowMenuItem
												icon={Delete02Icon}
												variant="destructive"
												href={routes.editTarget(target.id)}
											>
												{l.translate('menu.delete')}
											</RowMenuItem>
										{:else}
											<RowMenuItem
												icon={Delete02Icon}
												variant="destructive"
												disabled={appsUsing === undefined}
												onSelect={() => {
													removing = target;
													deleteOpen = true;
												}}
											>
												{l.translate('menu.delete')}
											</RowMenuItem>
										{/if}
									</RowMenuGroup>
								{/if}
							</RowMenu>
						</Table.Cell>
					</Table.Row>
				{:else}
					<Table.Row>
						<Table.Cell colspan={columns.shown.length + 1} class="text-muted-foreground">
							{l.translate('detail.list.no_match')}
						</Table.Cell>
					</Table.Row>
				{/each}
			</Table.Body>
		</Table.Root>
		<p class="text-[13px] text-muted-foreground">{l.translate('targets.explainer')}</p>
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

{#if removing}
	{@const target = removing}
	<ConfirmDialog
		bind:open={deleteOpen}
		destructive
		title={l.translate('targets.delete.title', [target.name])}
		description={target.state.last_ready_version
			? l.translate('target.delete.description', [target.name])
			: l.translate('targets.delete.never_ready')}
		confirmLabel={l.translate('target.delete')}
		icon={Delete02Icon}
		typeToConfirm={target.name}
		typeLabel={l.translate('target.delete.type_name')}
		onconfirm={remove}
	/>
{/if}
