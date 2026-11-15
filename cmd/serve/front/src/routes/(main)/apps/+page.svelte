<!--
	Applications (DESIGN.md §5 Tables, §11.1): one table row per application with its URL, target and
	the latest production and staging deployments. The name opens the application; the ⋯ menu has
	Open, Deploy, Logs, Settings, Copy URL and Delete. Filter by name and target, hide columns.
-->
<script lang="ts">
	import Add01Icon from '@hugeicons/core-free-icons/Add01Icon';
	import Clock01Icon from '@hugeicons/core-free-icons/Clock01Icon';
	import CommandLineIcon from '@hugeicons/core-free-icons/CommandLineIcon';
	import Copy01Icon from '@hugeicons/core-free-icons/Copy01Icon';
	import DashboardSquare01Icon from '@hugeicons/core-free-icons/DashboardSquare01Icon';
	import Delete02Icon from '@hugeicons/core-free-icons/Delete02Icon';
	import Globe02Icon from '@hugeicons/core-free-icons/Globe02Icon';
	import Rocket01Icon from '@hugeicons/core-free-icons/Rocket01Icon';
	import Search01Icon from '@hugeicons/core-free-icons/Search01Icon';
	import ServerStack01Icon from '@hugeicons/core-free-icons/ServerStack01Icon';
	import Settings02Icon from '@hugeicons/core-free-icons/Settings02Icon';
	import TestTube01Icon from '@hugeicons/core-free-icons/TestTube01Icon';
	import TextIcon from '@hugeicons/core-free-icons/TextIcon';
	import ViewIcon from '@hugeicons/core-free-icons/ViewIcon';
	import { toast } from 'svelte-sonner';
	import { copyText } from '#lib/clipboard.ts';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import EmptyState from '#lib/components/empty-state.svelte';
	import Icon from '#lib/components/icon.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import ColumnsMenu from '#lib/components/table/columns-menu.svelte';
	import FilterMenu from '#lib/components/table/filter-menu.svelte';
	import RowMenuGroup from '#lib/components/table/row-menu-group.svelte';
	import RowMenuItem from '#lib/components/table/row-menu-item.svelte';
	import RowMenu from '#lib/components/table/row-menu.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import * as Table from '#lib/components/ui/table/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service, { type App } from '#lib/resources/apps.ts';
	import { DeploymentStatus, type Deployment } from '#lib/resources/deployments.ts';
	import jobsService from '#lib/resources/jobs.ts';
	import { deploymentBadge, deploymentMoment } from '#lib/status.ts';
	import { ACTIONS_CELL, ACTIONS_HEAD, TableColumns } from '#lib/table-columns.svelte.ts';

	const { data: apps } = service.queryAll();
	const { data: jobs } = jobsService.queryAll({ page: 1 });

	const failed = $derived(
		($apps ?? []).filter(
			(app) => app.latest_deployments.production?.state.status === DeploymentStatus.Failed
		).length
	);

	const columns = new TableColumns(
		[
			{
				id: 'name',
				label: l.translate('column.name'),
				icon: TextIcon,
				width: 'w-[18%]',
				locked: true
			},
			{
				id: 'url',
				label: l.translate('column.url'),
				icon: Globe02Icon,
				width: 'w-[22%]',
				priority: 3
			},
			{
				id: 'target',
				label: l.translate('column.target'),
				icon: ServerStack01Icon,
				width: 'w-[14%]',
				priority: 2
			},
			{
				id: 'production',
				label: l.translate('column.production'),
				icon: Rocket01Icon,
				width: 'w-[17%]',
				locked: true
			},
			{
				id: 'staging',
				label: l.translate('column.staging'),
				icon: TestTube01Icon,
				width: 'w-[17%]',
				priority: 2
			},
			{
				id: 'updated',
				label: l.translate('column.updated'),
				icon: Clock01Icon,
				width: 'w-[12%]',
				priority: 3
			}
		],
		'apps'
	);

	let search = $state('');
	let target = $state('all');

	const targetOptions = $derived([
		{ value: 'all', label: l.translate('detail.filter.any'), icon: ServerStack01Icon },
		...[...new Map(($apps ?? []).map((a) => [a.production_target.id, a.production_target.name]))]
			.sort((a, b) => a[1].localeCompare(b[1]))
			.map(([value, label]) => ({ value, label, icon: ServerStack01Icon }))
	]);

	const shown = $derived(
		($apps ?? []).filter(
			(app) =>
				(target === 'all' ||
					app.production_target.id === target ||
					app.staging_target.id === target) &&
				app.name.toLowerCase().includes(search.trim().toLowerCase())
		)
	);

	/** The default service's URL: <app>.<target host> (how checkship exposes production). */
	function appUrl(app: App): string | undefined {
		if (!app.production_target.url) return undefined;
		try {
			const t = new URL(app.production_target.url);
			return `${t.protocol}//${app.name}.${t.host}`;
		} catch {
			return undefined;
		}
	}

	/** When the application last changed: its most recent deployment, else nothing. */
	function updated(app: App): string | undefined {
		const latest = [app.latest_deployments.production, app.latest_deployments.staging]
			.filter((d) => !!d)
			.map((d) => deploymentMoment(d))
			.sort()
			.at(-1);
		return latest ? l.relative(latest) : undefined;
	}

	async function copyUrl(url: string) {
		if (await copyText(url)) toast.success(l.translate('copied'));
		else toast.error(l.translate('copy_blocked'));
	}

	let deleting = $state<App>();
	let deleteOpen = $state(false);

	async function remove() {
		await service.delete(deleting!.id);
		await service.fetchAll({ cache: 'no-store' });
	}
</script>

{#snippet environment(deployment: Deployment | undefined)}
	{#if deployment}
		{@const badge = deploymentBadge(deployment.state)}
		<StatusBadge state={badge.state} label="{badge.label} · #{deployment.deployment_number}" />
	{:else}
		<span class="text-muted-foreground">{l.translate('apps.no_deployment')}</span>
	{/if}
{/snippet}

<PageHeader title={l.translate('breadcrumb.applications')}>
	{#snippet actions()}
		<Button href={routes.createApp}>
			<Icon icon={Add01Icon} />
			{l.translate('app.new')}
		</Button>
	{/snippet}
	{#snippet status()}
		{#if $apps}
			<span>{l.translate('apps.summary', [$apps.length, failed, $jobs?.total ?? 0])}</span>
		{/if}
	{/snippet}
</PageHeader>

{#if $apps && $apps.length > 0}
	<div class="flex flex-col gap-3">
		<div class="flex flex-wrap items-center justify-between gap-2">
			<div class="flex flex-wrap items-center gap-2">
				<div class="relative w-72 max-w-full">
					<Input
						type="search"
						bind:value={search}
						placeholder={l.translate('apps.filter')}
						aria-label={l.translate('apps.filter')}
						class="pr-9"
					/>
					<Icon
						icon={Search01Icon}
						class="pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2 text-muted-foreground"
					/>
				</div>
				<FilterMenu
					label={l.translate('column.target')}
					icon={ServerStack01Icon}
					bind:value={target}
					options={targetOptions}
				/>
			</div>
			<ColumnsMenu {columns} />
		</div>

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
				{#each shown as app (app.id)}
					{@const url = appUrl(app)}
					<Table.Row class="relative">
						{#if columns.visible('name')}
							<Table.Cell class={columns.cell('name')}>
								<!-- The whole row opens the application; the link stretches over it. -->
								<a
									href={routes.app(app.id)}
									title={app.name}
									class="text-sm font-semibold outline-none after:absolute after:inset-0 after:rounded-md focus-visible:after:ring-2 focus-visible:after:ring-ring focus-visible:after:ring-inset"
									>{app.name}</a
								>
							</Table.Cell>
						{/if}
						{#if columns.visible('url')}
							<Table.Cell
								class={columns.cell(
									'url',
									url ? 'font-mono text-[12.5px] text-primary' : 'text-muted-foreground'
								)}
								title={url}
							>
								{url ?? l.translate('targets.url.manual')}
							</Table.Cell>
						{/if}
						{#if columns.visible('target')}
							<Table.Cell class={columns.cell('target', 'text-muted-foreground')}>
								{app.production_target.name}
							</Table.Cell>
						{/if}
						{#if app.cleanup_requested_at}
							{#if columns.visible('production')}
								<Table.Cell class={columns.cell('production')}>
									<StatusBadge state="deleting" label={l.translate('apps.deletion_requested')} />
								</Table.Cell>
							{/if}
							{#if columns.visible('staging')}
								<Table.Cell class={columns.cell('staging', 'text-muted-foreground')}>—</Table.Cell>
							{/if}
						{:else}
							{#if columns.visible('production')}
								<Table.Cell class={columns.cell('production')}>
									{@render environment(app.latest_deployments.production)}
								</Table.Cell>
							{/if}
							{#if columns.visible('staging')}
								<Table.Cell class={columns.cell('staging')}>
									{@render environment(app.latest_deployments.staging)}
								</Table.Cell>
							{/if}
						{/if}
						{#if columns.visible('updated')}
							<Table.Cell class={columns.cell('updated', 'text-muted-foreground')}>
								{updated(app) ?? '—'}
							</Table.Cell>
						{/if}
						<Table.Cell class={ACTIONS_CELL}>
							<RowMenu label={l.translate('table.menu', [app.name])}>
								<RowMenuGroup>
									<RowMenuItem icon={ViewIcon} href={routes.app(app.id)}>
										{l.translate('menu.open')}
									</RowMenuItem>
									{#if !app.cleanup_requested_at}
										<RowMenuItem icon={Rocket01Icon} href={routes.createDeployment(app.id)}>
											{l.translate('menu.deploy')}
										</RowMenuItem>
										<RowMenuItem icon={CommandLineIcon} href={routes.appLogs(app.id)}>
											{l.translate('menu.logs')}
										</RowMenuItem>
										<RowMenuItem icon={Settings02Icon} href={routes.editApp(app.id)}>
											{l.translate('menu.settings')}
										</RowMenuItem>
									{/if}
									{#if url}
										<RowMenuItem icon={Copy01Icon} onSelect={() => copyUrl(url)}>
											{l.translate('menu.copy_url')}
										</RowMenuItem>
									{/if}
								</RowMenuGroup>
								{#if !app.cleanup_requested_at}
									<RowMenuGroup>
										<RowMenuItem
											icon={Delete02Icon}
											variant="destructive"
											onSelect={() => {
												deleting = app;
												deleteOpen = true;
											}}
										>
											{l.translate('menu.delete')}
										</RowMenuItem>
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
	</div>
{:else if $apps}
	<EmptyState
		icon={DashboardSquare01Icon}
		title={l.translate('apps.empty.title')}
		description={l.translate('apps.empty.description')}
	>
		{#snippet action()}
			<Button href={routes.createApp}>
				<Icon icon={Add01Icon} />
				{l.translate('app.new')}
			</Button>
		{/snippet}
	</EmptyState>
{/if}

{#if deleting}
	<ConfirmDialog
		bind:open={deleteOpen}
		destructive
		icon={Delete02Icon}
		title={l.translate('app.delete.title', [deleting.name])}
		description={l.translate('app.delete.description')}
		confirmLabel={l.translate('app.delete')}
		typeToConfirm={deleting.name}
		typeLabel={l.translate('app.delete.type_name', [deleting.name])}
		onconfirm={remove}
	/>
{/if}
