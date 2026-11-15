<!--
	Failed deployments of the period (opened from the Failures tile or a bar of the chart), newest
	first with their error code, paged with the activity cursor.
-->
<script lang="ts">
	import ArrowDown01Icon from '@hugeicons/core-free-icons/ArrowDown01Icon';
	import AlertCircleIcon from '@hugeicons/core-free-icons/AlertCircleIcon';
	import ArrowUpRight01Icon from '@hugeicons/core-free-icons/ArrowUpRight01Icon';
	import Clock01Icon from '@hugeicons/core-free-icons/Clock01Icon';
	import DashboardSquare01Icon from '@hugeicons/core-free-icons/DashboardSquare01Icon';
	import Layers01Icon from '@hugeicons/core-free-icons/Layers01Icon';
	import Rocket01Icon from '@hugeicons/core-free-icons/Rocket01Icon';
	import ViewIcon from '@hugeicons/core-free-icons/ViewIcon';
	import { untrack } from 'svelte';
	import Drawer from '#lib/components/drawer.svelte';
	import EnvironmentChip from '#lib/components/environment-chip.svelte';
	import Icon from '#lib/components/icon.svelte';
	import RowMenuGroup from '#lib/components/table/row-menu-group.svelte';
	import RowMenuItem from '#lib/components/table/row-menu-item.svelte';
	import RowMenu from '#lib/components/table/row-menu.svelte';
	import * as Table from '#lib/components/ui/table/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { appendFeed, deploymentKey } from '#lib/dashboard.ts';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service, { type DashboardFilters } from '#lib/resources/dashboard.ts';
	import { DeploymentStatus, type Deployment } from '#lib/resources/deployments.ts';
	import { ACTIONS_CELL, ACTIONS_HEAD, TableColumns } from '#lib/table-columns.svelte.ts';

	type Props = {
		open: boolean;
		filters: DashboardFilters;
		appName: (id: string) => string | undefined;
	};

	let { open = $bindable(false), filters, appName }: Props = $props();

	let rows = $state<Deployment[]>([]);
	let cursor = $state<string | null>(null);
	let loading = $state(false);
	let error = $state(false);

	async function load(more: boolean) {
		loading = true;
		error = false;
		try {
			const page = await service.fetchActivity({
				range: filters.range,
				environment: filters.environment,
				status: DeploymentStatus.Failed,
				limit: 10,
				cursor: more ? (cursor ?? undefined) : undefined
			});
			rows = more ? appendFeed(rows, page.data) : page.data;
			cursor = page.next_cursor;
		} catch {
			error = true;
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		if (!open) return;
		void filters.range;
		void filters.environment;
		untrack(() => load(false));
	});

	const columns = new TableColumns([
		{
			id: 'deployment',
			label: l.translate('column.deployment'),
			icon: Rocket01Icon,
			width: 'w-[30%]'
		},
		{
			id: 'environment',
			label: l.translate('column.environment'),
			icon: Layers01Icon,
			width: 'w-[20%]',
			priority: 2
		},
		{ id: 'error', label: l.translate('column.error'), icon: AlertCircleIcon, width: 'w-[28%]' },
		{
			id: 'started',
			label: l.translate('column.started'),
			icon: Clock01Icon,
			width: 'w-[22%]',
			priority: 3
		}
	]);

	const format = new Intl.DateTimeFormat(l.locale(), { dateStyle: 'medium', timeStyle: 'short' });
</script>

<Drawer
	bind:open
	title={l.translate('dashboard.failures.title')}
	description={l.translate('dashboard.failures.description', [
		filters.range,
		filters.environment
			? l.translate(`deploy.environment.${filters.environment}`)
			: l.translate('dashboard.environment.all')
	])}
>
	{#if error}
		<p class="text-sm text-danger">{l.translate('dashboard.tile.error')}</p>
	{:else if rows.length === 0 && !loading}
		<p class="text-sm text-muted-foreground">{l.translate('dashboard.failures.none')}</p>
	{:else}
		<Table.Root aria-label={l.translate('dashboard.failures.title')}>
			<Table.Header>
				<Table.Row>
					{#each columns.shown as column (column.id)}
						<Table.Head class={columns.head(column.id)}>{column.label}</Table.Head>
					{/each}
					<Table.Head class={ACTIONS_HEAD}>{l.translate('column.actions')}</Table.Head>
				</Table.Row>
			</Table.Header>
			<Table.Body>
				{#each rows as d (deploymentKey(d))}
					{@const name = `${appName(d.app_id) ?? d.app_id} #${d.deployment_number}`}
					<Table.Row class="relative">
						<Table.Cell class={columns.cell('deployment')}>
							<a
								href={routes.deployment(d.app_id, d.deployment_number)}
								title={name}
								class="text-sm font-semibold outline-none after:absolute after:inset-0 after:rounded-md focus-visible:after:ring-2 focus-visible:after:ring-ring focus-visible:after:ring-inset"
								>{name}</a
							>
						</Table.Cell>
						<Table.Cell class={columns.cell('environment')}>
							<EnvironmentChip environment={d.environment} />
						</Table.Cell>
						<Table.Cell
							class={columns.cell('error', 'font-mono text-[12.5px] text-danger')}
							title={d.state.error_code}
						>
							{d.state.error_code ?? '—'}
						</Table.Cell>
						<Table.Cell class={columns.cell('started', 'text-muted-foreground')}>
							{format.format(new Date(d.requested_at))}
						</Table.Cell>
						<Table.Cell class={ACTIONS_CELL}>
							<RowMenu label={l.translate('table.menu', [name])}>
								<RowMenuGroup>
									<RowMenuItem
										icon={ViewIcon}
										href={routes.deployment(d.app_id, d.deployment_number)}
									>
										{l.translate('jobs.open.deployment')}
									</RowMenuItem>
									<RowMenuItem icon={DashboardSquare01Icon} href={routes.app(d.app_id)}>
										{l.translate('jobs.open.app')}
									</RowMenuItem>
								</RowMenuGroup>
							</RowMenu>
						</Table.Cell>
					</Table.Row>
				{/each}
			</Table.Body>
		</Table.Root>
		{#if cursor}
			<div class="flex justify-center">
				<Button variant="outline" size="sm" {loading} onclick={() => load(true)}>
					<Icon icon={ArrowDown01Icon} />
					{l.translate('dashboard.load_more')}
				</Button>
			</div>
		{/if}
	{/if}

	{#snippet footer()}
		<Button variant="outline" href={routes.jobs} class="mr-auto">
			<Icon icon={ArrowUpRight01Icon} />
			{l.translate('dashboard.failures.jobs')}
		</Button>
		<Button variant="outline" onclick={() => (open = false)}>{l.translate('close')}</Button>
	{/snippet}
</Drawer>
