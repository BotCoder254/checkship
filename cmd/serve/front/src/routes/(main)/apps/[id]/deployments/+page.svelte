<!--
	Deployments tab (DESIGN.md §11.7): the whole history, filtered on the server by environment,
	status and a search on number, branch or commit. A list grouped by day (§5 Lists and tables),
	20 at a time with Show more. Each row opens its preview; its menu (Railway) has View logs,
	Promote (staging rows), Redeploy and Copy link. The deployment each environment serves is marked
	Live.
-->
<script lang="ts">
	import { copyText } from '#lib/clipboard.ts';
	import CancelCircleIcon from '@hugeicons/core-free-icons/CancelCircleIcon';
	import CheckmarkCircle02Icon from '@hugeicons/core-free-icons/CheckmarkCircle02Icon';
	import ArrowDown01Icon from '@hugeicons/core-free-icons/ArrowDown01Icon';
	import CommandLineIcon from '@hugeicons/core-free-icons/CommandLineIcon';
	import Copy01Icon from '@hugeicons/core-free-icons/Copy01Icon';
	import Layers01Icon from '@hugeicons/core-free-icons/Layers01Icon';
	import Loading03Icon from '@hugeicons/core-free-icons/Loading03Icon';
	import RepeatIcon from '@hugeicons/core-free-icons/RepeatIcon';
	import Rocket01Icon from '@hugeicons/core-free-icons/Rocket01Icon';
	import Search01Icon from '@hugeicons/core-free-icons/Search01Icon';
	import TestTube01Icon from '@hugeicons/core-free-icons/TestTube01Icon';
	import ViewIcon from '@hugeicons/core-free-icons/ViewIcon';
	import { toast } from 'svelte-sonner';
	import { fromStore } from 'svelte/store';
	import TimelineGroup from '#lib/components/timeline/timeline-group.svelte';
	import TimelineRow from '#lib/components/timeline/timeline-row.svelte';
	import {
		clock,
		deploymentCommit,
		deploymentDuration,
		deploymentLabel,
		deploymentSource,
		deploymentTile
	} from '#lib/deployment-row.ts';
	import { deploymentMoment } from '#lib/status.ts';
	import { groupByDay } from '#lib/timeline.ts';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import EmptyState from '#lib/components/empty-state.svelte';
	import Icon from '#lib/components/icon.svelte';
	import EnvironmentChip from '#lib/components/environment-chip.svelte';
	import FilterMenu, { type FilterOption } from '#lib/components/table/filter-menu.svelte';
	import RowMenuGroup from '#lib/components/table/row-menu-group.svelte';
	import RowMenuItem from '#lib/components/table/row-menu-item.svelte';
	import RowMenu from '#lib/components/table/row-menu.svelte';
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
	/** How many pages of 20 are shown; Show more adds one. */
	let pages = $state(1);

	// Search after a short pause in typing; any filter change goes back to the first 20.
	$effect(() => {
		const value = search.trim();
		const timer = setTimeout(() => (searched = value), 300);
		return () => clearTimeout(timer);
	});
	$effect(() => {
		void environment;
		void status;
		void searched;
		pages = 1;
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
		const f: QueryDeploymentsFilters = { per_page: PER_PAGE };
		if (environment !== 'all') f.environment = environment;
		if (statusValue[status] !== undefined) f.status = statusValue[status];
		if (searched) f.search = searched;
		return f;
	});
	// One polled query per page shown (the server caps a page at 50), joined in order. A deployment
	// queued meanwhile shifts the pages by one, so a row can come back twice: keep the first.
	const pageQueries = $derived(
		Array.from({ length: pages }, (_, i) =>
			fromStore(service.queryAllByApp(data.app.id, { ...filters, page: i + 1 }).data)
		)
	);
	const first = $derived(pageQueries[0].current);
	const rows = $derived.by(() => {
		const seen: Record<number, true> = {};
		return pageQueries
			.flatMap((q) => q.current?.data ?? [])
			.filter((d) => !seen[d.deployment_number] && (seen[d.deployment_number] = true));
	});
	const total = $derived(first?.total ?? 0);
	const groups = $derived(groupByDay(rows, deploymentMoment, l.locale()));
	const loadingMore = $derived(pages > 1 && !pageQueries[pages - 1].current);

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
	{#snippet children(workspace: AppActions)}
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
			</div>

			{#if rows.length > 0}
				<div class="flex flex-col gap-4">
					{#each groups as group (group.key)}
						<TimelineGroup label={group.label} count={group.items.length}>
							{#each group.items as d (d.deployment_number)}
								{@const tile = deploymentTile(d.state)}
								{@const live = isLive(d)}
								{@const duration = deploymentDuration(d)}
								<TimelineRow
									icon={tile.icon}
									tone={tile.tone}
									spin={tile.spin}
									label={deploymentLabel(d, live ? l.translate('column.live') : undefined)}
									selected={workspace.previewNumber() === d.deployment_number}
									onclick={() => workspace.openPreview(d.deployment_number)}
								>
									{#snippet title()}#{d.deployment_number}{/snippet}
									{#snippet badge()}
										<EnvironmentChip environment={d.environment} />
										{#if live}
											<span
												class="inline-flex h-5 items-center gap-1.5 rounded-sm bg-success-soft px-1.5 text-xs font-semibold"
											>
												<span class="size-1.5 rounded-full bg-success"></span>
												{l.translate('column.live')}
											</span>
										{/if}
									{/snippet}
									{#snippet details()}
										{clock(deploymentMoment(d))}
										{#if deploymentSource(d.source)}
											· <span class={d.source.discriminator === 'git' ? 'font-mono' : ''}
												>{deploymentSource(d.source)}</span
											>
										{/if}
										{#if d.state.error_code}
											· <span class="font-mono text-danger">{d.state.error_code}</span>
										{:else if deploymentCommit(d.source)}
											· <span class="font-mono">{deploymentCommit(d.source)}</span>
										{/if}
										{#if duration}· {duration}{/if}
										· {d.requested_by.email}
									{/snippet}
									{#snippet actions()}
										<RowMenu label={l.translate('detail.row.menu', [d.deployment_number])}>
											<RowMenuGroup>
												<RowMenuItem
													icon={CommandLineIcon}
													onSelect={() => workspace.openPreview(d.deployment_number)}
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
									{/snippet}
								</TimelineRow>
							{/each}
						</TimelineGroup>
					{/each}

					<div class="flex flex-wrap items-center gap-3 text-label text-muted-foreground">
						{#if rows.length < total}
							<Button variant="outline" size="sm" loading={loadingMore} onclick={() => pages++}>
								<Icon icon={ArrowDown01Icon} />
								{l.translate('list.show_more', [Math.min(PER_PAGE, total - rows.length)])}
							</Button>
						{/if}
						<span class="tabular-nums"
							>{l.translate('detail.list.count', [rows.length, total])}</span
						>
					</div>
				</div>
			{:else if first}
				<EmptyState
					illustration={filtered ? 'search' : 'rocket'}
					size={filtered ? 'section' : 'page'}
					title={filtered ? l.translate('detail.list.no_match') : l.translate('app.empty.title')}
					description={filtered
						? l.translate('detail.list.no_match.description')
						: l.translate('app.empty.description')}
				>
					{#snippet action()}
						{#if filtered}
							<Button
								variant="outline"
								onclick={() => {
									environment = 'all';
									status = 'any';
									search = '';
								}}
							>
								{l.translate('list.clear_filters')}
							</Button>
						{:else if !app.cleanup_requested_at}
							<Button onclick={() => workspace.openDeploy('production')}>
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
					.then((d) => workspace.openPreview(d.deployment_number))}
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
						.then((d) => workspace.openPreview(d.deployment_number))}
			/>
		{/if}
	{/snippet}
</AppWorkspace>
