<!--
	Deployments tab (DESIGN.md §11.7): the whole history, filtered on the server by environment,
	status and a search on number, branch or commit. Each row opens its preview; its menu (Railway)
	has View logs, Promote (staging rows), Redeploy and Copy link. The deployment each environment
	serves is marked Live.
-->
<script lang="ts">
	import { copyText } from '#lib/clipboard.ts';
	import ArrowLeft01Icon from '@hugeicons/core-free-icons/ArrowLeft01Icon';
	import ArrowRight01Icon from '@hugeicons/core-free-icons/ArrowRight01Icon';
	import CancelCircleIcon from '@hugeicons/core-free-icons/CancelCircleIcon';
	import CheckmarkCircle02Icon from '@hugeicons/core-free-icons/CheckmarkCircle02Icon';
	import Clock01Icon from '@hugeicons/core-free-icons/Clock01Icon';
	import CommandLineIcon from '@hugeicons/core-free-icons/CommandLineIcon';
	import Copy01Icon from '@hugeicons/core-free-icons/Copy01Icon';
	import Layers01Icon from '@hugeicons/core-free-icons/Layers01Icon';
	import Loading03Icon from '@hugeicons/core-free-icons/Loading03Icon';
	import RepeatIcon from '@hugeicons/core-free-icons/RepeatIcon';
	import Rocket01Icon from '@hugeicons/core-free-icons/Rocket01Icon';
	import Search01Icon from '@hugeicons/core-free-icons/Search01Icon';
	import GitBranchIcon from '@hugeicons/core-free-icons/GitBranchIcon';
	import GitCommitIcon from '@hugeicons/core-free-icons/GitCommitIcon';
	import SourceCodeIcon from '@hugeicons/core-free-icons/SourceCodeIcon';
	import Tag01Icon from '@hugeicons/core-free-icons/Tag01Icon';
	import TestTube01Icon from '@hugeicons/core-free-icons/TestTube01Icon';
	import Timer02Icon from '@hugeicons/core-free-icons/Timer02Icon';
	import UserIcon from '@hugeicons/core-free-icons/UserIcon';
	import ViewIcon from '@hugeicons/core-free-icons/ViewIcon';
	import { toast } from 'svelte-sonner';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import EmptyState from '#lib/components/empty-state.svelte';
	import Icon from '#lib/components/icon.svelte';
	import EnvironmentChip from '#lib/components/environment-chip.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import ColumnsMenu from '#lib/components/table/columns-menu.svelte';
	import FilterMenu, { type FilterOption } from '#lib/components/table/filter-menu.svelte';
	import RowMenuGroup from '#lib/components/table/row-menu-group.svelte';
	import RowMenuItem from '#lib/components/table/row-menu-item.svelte';
	import RowMenu from '#lib/components/table/row-menu.svelte';
	import * as Table from '#lib/components/ui/table/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Input } from '#lib/components/ui/input/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import appsService from '#lib/resources/apps.ts';
	import service, {
		DeploymentStatus,
		type Deployment,
		type QueryDeploymentsFilters
	} from '#lib/resources/deployments.ts';
	import { deploymentBadge, deploymentMoment } from '#lib/status.ts';
	import { ACTIONS_CELL, ACTIONS_HEAD, TableColumns } from '#lib/table-columns.svelte.ts';
	import AppWorkspace, { type AppActions } from '../app-workspace.svelte';

	let { data } = $props();

	const appQuery = $derived(appsService.queryById(data.app.id));
	const appStore = $derived(appQuery.data);
	const app = $derived($appStore ?? data.app);

	const PER_PAGE = 20;

	let environment = $state<'all' | 'production' | 'staging'>('all');
	let status = $state<'any' | 'succeeded' | 'failed' | 'running'>('any');
	let search = $state('');
	let searched = $state('');
	let page = $state(1);

	// Search after a short pause in typing; any filter change goes back to page 1.
	$effect(() => {
		const value = search.trim();
		const timer = setTimeout(() => (searched = value), 300);
		return () => clearTimeout(timer);
	});
	$effect(() => {
		void environment;
		void status;
		void searched;
		page = 1;
	});

	const environmentLabel: Record<typeof environment, string> = {
		all: l.translate('detail.filter.all_environments'),
		production: l.translate('deploy.environment.production'),
		staging: l.translate('deploy.environment.staging')
	};
	const statusLabel: Record<typeof status, string> = {
		any: l.translate('detail.filter.any'),
		succeeded: l.translate('status.succeeded'),
		failed: l.translate('status.failed'),
		running: l.translate('status.running')
	};

	const statusValue: Record<typeof status, DeploymentStatus | undefined> = {
		any: undefined,
		succeeded: DeploymentStatus.Succeeded,
		failed: DeploymentStatus.Failed,
		running: DeploymentStatus.Running
	};

	// Only send what is set: unset values would reach the server as the text "undefined".
	const filters = $derived.by(() => {
		const f: QueryDeploymentsFilters = { page, per_page: PER_PAGE };
		if (environment !== 'all') f.environment = environment;
		if (statusValue[status] !== undefined) f.status = statusValue[status];
		if (searched) f.search = searched;
		return f;
	});
	const listQuery = $derived(service.queryAllByApp(data.app.id, filters));
	const listStore = $derived(listQuery.data);
	const list = $derived($listStore);

	const filtered = $derived(environment !== 'all' || status !== 'any' || !!searched);

	const isLive = (d: Deployment) =>
		d.state.status === DeploymentStatus.Succeeded &&
		app.latest_deployments[d.environment]?.deployment_number === d.deployment_number;

	let promoteNumber = $state(0);
	let promoteOpen = $state(false);
	let redeploy = $state<Deployment>();
	let redeployOpen = $state(false);

	async function copyLink(d: Deployment) {
		const link = new URL(routes.deployment(app.id, d.deployment_number), location.origin).href;
		if (await copyText(link)) toast.success(l.translate('copied'));
		else toast.error(l.translate('copy_blocked'));
	}

	const columns = new TableColumns(
		[
			{
				id: 'status',
				label: l.translate('column.status'),
				icon: CheckmarkCircle02Icon,
				width: 'w-[12%]',
				locked: true
			},
			{
				id: 'number',
				label: l.translate('column.number'),
				icon: Tag01Icon,
				width: 'w-[8%]',
				locked: true
			},
			{
				id: 'environment',
				label: l.translate('column.environment'),
				icon: Layers01Icon,
				width: 'w-[12%]'
			},
			{
				id: 'source',
				label: l.translate('column.source'),
				icon: SourceCodeIcon,
				width: 'w-[10%]',
				priority: 3
			},
			{
				id: 'branch',
				label: l.translate('column.branch'),
				icon: GitBranchIcon,
				width: 'w-[9%]',
				priority: 2
			},
			{
				id: 'commit',
				label: l.translate('column.commit'),
				icon: GitCommitIcon,
				width: 'w-[11%]',
				priority: 2
			},
			{
				id: 'author',
				label: l.translate('column.author'),
				icon: UserIcon,
				width: 'w-[14%]',
				priority: 3
			},
			{ id: 'started', label: l.translate('column.started'), icon: Clock01Icon, width: 'w-[13%]' },
			{
				id: 'duration',
				label: l.translate('column.duration'),
				icon: Timer02Icon,
				width: 'w-[9%]',
				priority: 3
			}
		],
		'deployments'
	);

	const environmentOptions: FilterOption<typeof environment>[] = [
		{ value: 'all', label: environmentLabel.all, icon: Layers01Icon },
		{ value: 'production', label: environmentLabel.production, icon: Rocket01Icon },
		{ value: 'staging', label: environmentLabel.staging, icon: TestTube01Icon }
	];
	const statusOptions: FilterOption<typeof status>[] = [
		{ value: 'any', label: statusLabel.any, icon: Layers01Icon },
		{
			value: 'succeeded',
			label: statusLabel.succeeded,
			icon: CheckmarkCircle02Icon,
			tone: 'text-success'
		},
		{ value: 'failed', label: statusLabel.failed, icon: CancelCircleIcon, tone: 'text-danger' },
		{ value: 'running', label: statusLabel.running, icon: Loading03Icon, tone: 'text-warning' }
	];
</script>

<AppWorkspace {app} apiKey={data.user.api_key} tab="deployments">
	{#snippet children(actions: AppActions)}
		<div class="flex flex-col gap-3.5">
			<div class="flex flex-wrap items-center justify-between gap-2">
				<div class="flex flex-wrap items-center gap-2">
					<div class="relative w-80 max-w-full">
						<Input
							id="filter-search"
							type="search"
							bind:value={search}
							placeholder={l.translate('detail.filter.search')}
							aria-label={l.translate('detail.filter.search.label')}
							class="pr-9"
						/>
						<Icon
							icon={Search01Icon}
							class="pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2 text-muted-foreground"
						/>
					</div>
					<FilterMenu
						label={l.translate('column.environment')}
						icon={Layers01Icon}
						bind:value={environment}
						options={environmentOptions}
					/>
					<FilterMenu
						label={l.translate('column.status')}
						icon={CheckmarkCircle02Icon}
						bind:value={status}
						options={statusOptions}
					/>
				</div>
				<ColumnsMenu {columns} />
			</div>

			{#if list && list.data.length > 0}
				<div class="flex flex-col gap-2">
					<Table.Root aria-label={l.translate('detail.tab.deployments')}>
						<Table.Header>
							<Table.Row>
								{#each columns.shown as column (column.id)}
									<Table.Head class={columns.head(column.id)}>{column.label}</Table.Head>
								{/each}
								<Table.Head class={ACTIONS_HEAD}>{l.translate('column.actions')}</Table.Head>
							</Table.Row>
						</Table.Header>
						<Table.Body>
							{#each list.data as d (d.deployment_number)}
								{@const badge = deploymentBadge(d.state)}
								{@const live = isLive(d)}
								{@const hash = d.source.discriminator === 'git' ? d.source.data.hash : undefined}
								<Table.Row
									class="relative"
									data-state={actions.previewNumber() === d.deployment_number
										? 'selected'
										: undefined}
								>
									{#if columns.visible('status')}
										<Table.Cell class={columns.cell('status')}>
											<StatusBadge state={badge.state} label={badge.label} />
										</Table.Cell>
									{/if}
									{#if columns.visible('number')}
										<Table.Cell class={columns.cell('number', 'font-semibold tabular-nums')}>
											<!-- The whole row opens the preview; the button stretches over it. -->
											<button
												type="button"
												class="outline-none after:absolute after:inset-0 after:rounded-md focus-visible:after:ring-2 focus-visible:after:ring-ring focus-visible:after:ring-inset"
												onclick={() => actions.openPreview(d.deployment_number)}
												>#{d.deployment_number}</button
											>
											{#if live}
												<span
													class="ml-1 inline-block size-2 rounded-full bg-success"
													title={l.translate('column.live')}
												></span>
												<span class="sr-only">{l.translate('column.live')}</span>
											{/if}
										</Table.Cell>
									{/if}
									{#if columns.visible('environment')}
										<Table.Cell class={columns.cell('environment')}>
											<EnvironmentChip environment={d.environment} />
										</Table.Cell>
									{/if}
									{#if columns.visible('source')}
										<Table.Cell
											class={columns.cell('source', 'text-muted-foreground')}
											title={l.translate(`deployment.source.${d.source.discriminator}`)}
										>
											{l.translate(`deployment.source.${d.source.discriminator}`)}
										</Table.Cell>
									{/if}
									{#if columns.visible('branch')}
										<Table.Cell
											class={columns.cell('branch')}
											title={d.source.discriminator === 'git' ? d.source.data.branch : undefined}
										>
											{d.source.discriminator === 'git' ? d.source.data.branch : '—'}
										</Table.Cell>
									{/if}
									{#if columns.visible('commit')}
										<Table.Cell
											class={columns.cell(
												'commit',
												d.state.error_code
													? 'font-mono text-[12.5px] text-danger'
													: 'font-mono text-[12.5px] text-muted-foreground'
											)}
											title={d.state.error_code ?? hash}
										>
											{d.state.error_code ?? hash?.slice(0, 7) ?? '—'}
										</Table.Cell>
									{/if}
									{#if columns.visible('author')}
										<Table.Cell
											class={columns.cell('author', 'text-muted-foreground')}
											title={d.requested_by.email}
										>
											{d.requested_by.email}
										</Table.Cell>
									{/if}
									{#if columns.visible('started')}
										<Table.Cell
											class={columns.cell('started', 'text-muted-foreground tabular-nums')}
										>
											{l.relative(deploymentMoment(d))}
										</Table.Cell>
									{/if}
									{#if columns.visible('duration')}
										<Table.Cell
											class={columns.cell('duration', 'text-muted-foreground tabular-nums')}
										>
											{d.state.started_at && d.state.finished_at
												? l.duration(d.state.started_at, d.state.finished_at)
												: '—'}
										</Table.Cell>
									{/if}
									<Table.Cell class={ACTIONS_CELL}>
										<RowMenu label={l.translate('detail.row.menu', [d.deployment_number])}>
											<RowMenuGroup>
												<RowMenuItem
													icon={CommandLineIcon}
													onSelect={() => actions.openPreview(d.deployment_number)}
												>
													{l.translate('detail.row.logs')}
												</RowMenuItem>
												<RowMenuItem
													icon={ViewIcon}
													href={routes.deployment(app.id, d.deployment_number)}
												>
													{l.translate('menu.open')}
												</RowMenuItem>
												{#if !app.cleanup_requested_at}
													{#if d.environment === 'staging' && d.state.status === DeploymentStatus.Succeeded}
														<RowMenuItem
															icon={Rocket01Icon}
															onSelect={() => {
																promoteNumber = d.deployment_number;
																promoteOpen = true;
															}}
														>
															{l.translate('detail.row.promote')}
														</RowMenuItem>
													{/if}
													{#if d.state.finished_at}
														<RowMenuItem
															icon={RepeatIcon}
															onSelect={() => {
																redeploy = d;
																redeployOpen = true;
															}}
														>
															{l.translate('deployment.redeploy')}
														</RowMenuItem>
													{/if}
												{/if}
												<RowMenuItem icon={Copy01Icon} onSelect={() => copyLink(d)}>
													{l.translate('detail.row.copy')}
												</RowMenuItem>
											</RowMenuGroup>
										</RowMenu>
									</Table.Cell>
								</Table.Row>
							{/each}
						</Table.Body>
					</Table.Root>

					<div class="flex items-center justify-between pt-2 text-[13px] text-muted-foreground">
						<span>
							{l.translate('detail.list.summary', [
								list.data.length,
								list.total,
								list.page,
								Math.max(1, Math.ceil(list.total / list.per_page))
							])}
						</span>
						{#if !(list.first_page && list.last_page)}
							<span class="flex gap-1">
								<Button
									variant="outline"
									size="icon-sm"
									aria-label={l.translate('pagination.previous')}
									disabled={list.first_page}
									onclick={() => page--}
								>
									<Icon icon={ArrowLeft01Icon} />
								</Button>
								<Button
									variant="outline"
									size="icon-sm"
									aria-label={l.translate('pagination.next')}
									disabled={list.last_page}
									onclick={() => page++}
								>
									<Icon icon={ArrowRight01Icon} />
								</Button>
							</span>
						{/if}
					</div>
				</div>
			{:else if list}
				<EmptyState
					icon={Layers01Icon}
					title={filtered ? l.translate('detail.list.no_match') : l.translate('app.empty.title')}
					description={filtered
						? l.translate('detail.list.no_match.description')
						: l.translate('app.empty.description')}
				>
					{#snippet action()}
						{#if !filtered && !app.cleanup_requested_at}
							<Button onclick={() => actions.openDeploy('production')}>
								<Icon icon={Rocket01Icon} />
								{l.translate('deployment.new')}
							</Button>
						{/if}
					{/snippet}
				</EmptyState>
			{/if}
		</div>

		<ConfirmDialog
			bind:open={promoteOpen}
			title={l.translate('deployment.promote.title')}
			description={l.translate('deployment.promote.description', [promoteNumber])}
			confirmLabel={l.translate('deployment.promote.title')}
			icon={Rocket01Icon}
			onconfirm={() =>
				service
					.promote(app.id, promoteNumber)
					.then((d) => actions.openPreview(d.deployment_number))}
		/>
		{#if redeploy}
			{@const target = redeploy}
			<ConfirmDialog
				bind:open={redeployOpen}
				title={l.translate('deployment.redeploy')}
				description={l.translate('deployment.redeploy.description', [
					target.deployment_number,
					target.environment
				])}
				confirmLabel={l.translate('deployment.redeploy')}
				icon={RepeatIcon}
				onconfirm={() =>
					service
						.redeploy(app.id, target.deployment_number)
						.then((d) => actions.openPreview(d.deployment_number))}
			/>
		{/if}
	{/snippet}
</AppWorkspace>
