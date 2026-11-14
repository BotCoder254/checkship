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
	import MoreHorizontalIcon from '@hugeicons/core-free-icons/MoreHorizontalIcon';
	import RepeatIcon from '@hugeicons/core-free-icons/RepeatIcon';
	import Rocket01Icon from '@hugeicons/core-free-icons/Rocket01Icon';
	import Search01Icon from '@hugeicons/core-free-icons/Search01Icon';
	import { toast } from 'svelte-sonner';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import EmptyState from '#lib/components/empty-state.svelte';
	import type { IconData } from '#lib/components/icon.svelte';
	import Icon from '#lib/components/icon.svelte';
	import { Label } from '#lib/components/ui/label/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as DropdownMenu from '#lib/components/ui/dropdown-menu/index.js';
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
	import { cn } from '#lib/utils.js';
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

	const rowIcon: Record<DeploymentStatus, { icon: IconData; tone: string; spin?: boolean }> = {
		[DeploymentStatus.Succeeded]: { icon: CheckmarkCircle02Icon, tone: 'text-success' },
		[DeploymentStatus.Failed]: { icon: CancelCircleIcon, tone: 'text-danger' },
		[DeploymentStatus.Running]: { icon: Loading03Icon, tone: 'text-warning', spin: true },
		[DeploymentStatus.Pending]: { icon: Clock01Icon, tone: 'text-muted-foreground' }
	};

	const isLive = (d: Deployment) =>
		d.state.status === DeploymentStatus.Succeeded &&
		app.latest_deployments[d.environment]?.deployment_number === d.deployment_number;

	function timing(d: Deployment) {
		const { started_at, finished_at } = d.state;
		const when = l.relative(deploymentMoment(d));
		return started_at && finished_at ? `${when} · ${l.duration(started_at, finished_at)}` : when;
	}

	let promoteNumber = $state(0);
	let promoteOpen = $state(false);
	let redeploy = $state<Deployment>();
	let redeployOpen = $state(false);

	async function copyLink(d: Deployment) {
		const link = new URL(routes.deployment(app.id, d.deployment_number), location.origin).href;
		if (await copyText(link)) toast.success(l.translate('copied'));
		else toast.error(l.translate('copy_blocked'));
	}

	const columns = 'md:grid-cols-[124px_48px_100px_minmax(0,1fr)_minmax(0,170px)_140px_56px_36px]';
</script>

<AppWorkspace {app} apiKey={data.user.api_key} tab="deployments">
	{#snippet children(actions: AppActions)}
		<div class="flex flex-col gap-3.5">
			<!-- Search with a readable fixed width, then the two filters as dropdowns. -->
			<div class="grid items-end gap-3 sm:grid-cols-[minmax(0,320px)_190px_190px]">
				<div class="flex flex-col gap-1">
					<Label for="filter-search" class="text-xs font-semibold text-muted-foreground">
						{l.translate('detail.filter.search.label')}
					</Label>
					<div class="relative">
						<Input
							id="filter-search"
							type="search"
							bind:value={search}
							placeholder={l.translate('detail.filter.search')}
							class="pr-9"
						/>
						<Icon
							icon={Search01Icon}
							class="pointer-events-none absolute top-1/2 right-3 size-4 -translate-y-1/2 text-muted-foreground"
						/>
					</div>
				</div>
				<div class="flex flex-col gap-1">
					<Label for="filter-env" class="text-xs font-semibold text-muted-foreground">
						{l.translate('deployment.environment')}
					</Label>
					<Select.Root type="single" bind:value={environment}>
						<Select.Trigger id="filter-env" class="w-full">
							{environmentLabel[environment]}
						</Select.Trigger>
						<Select.Content>
							{#each Object.entries(environmentLabel) as [value, label] (value)}
								<Select.Item {value} {label}>{label}</Select.Item>
							{/each}
						</Select.Content>
					</Select.Root>
				</div>
				<div class="flex flex-col gap-1">
					<Label for="filter-status" class="text-xs font-semibold text-muted-foreground">
						{l.translate('jobs.status')}
					</Label>
					<Select.Root type="single" bind:value={status}>
						<Select.Trigger id="filter-status" class="w-full">
							{statusLabel[status]}
						</Select.Trigger>
						<Select.Content>
							{#each Object.entries(statusLabel) as [value, label] (value)}
								<Select.Item {value} {label}>{label}</Select.Item>
							{/each}
						</Select.Content>
					</Select.Root>
				</div>
			</div>

			{#if list && list.data.length > 0}
				<div
					role="table"
					aria-label={l.translate('detail.tab.deployments')}
					class="flex flex-col gap-1.5"
				>
					<div
						role="row"
						class={cn(
							'hidden gap-3 pr-2 pb-0.5 pl-4 text-xs font-semibold text-muted-foreground md:grid',
							columns
						)}
					>
						<span role="columnheader">{l.translate('jobs.status')}</span>
						<span role="columnheader">#</span>
						<span role="columnheader">{l.translate('deployment.environment')}</span>
						<span role="columnheader">{l.translate('deployment.source')}</span>
						<span role="columnheader">{l.translate('deployment.queued_by')}</span>
						<span role="columnheader">{l.translate('detail.column.when')}</span>
						<span role="columnheader"
							><span class="sr-only">{l.translate('detail.card.live')}</span></span
						>
						<span role="columnheader"
							><span class="sr-only">{l.translate('detail.row.actions')}</span></span
						>
					</div>
					{#each list.data as d (d.deployment_number)}
						{@const row = rowIcon[d.state.status]}
						{@const live = isLive(d)}
						<div
							role="row"
							class={cn(
								'relative grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-1 rounded-md bg-sunken py-2.5 pr-2 pl-4 text-[13px] transition-colors hover:bg-hover md:min-h-13 md:py-0',
								columns,
								actions.previewNumber() === d.deployment_number && 'bg-raised hover:bg-raised'
							)}
						>
							<span role="cell" class="flex items-center gap-2">
								<Icon icon={row.icon} class={cn('size-4', row.tone, row.spin && 'animate-spin')} />
								<!-- The whole row opens the preview; the button stretches over it. -->
								<button
									type="button"
									class="text-left outline-none after:absolute after:inset-0 after:rounded-md focus-visible:after:ring-2 focus-visible:after:ring-ring focus-visible:after:ring-inset"
									onclick={() => actions.openPreview(d.deployment_number)}
								>
									{deploymentBadge(d.state).label}
									<span class="font-semibold md:hidden">#{d.deployment_number}</span>
								</button>
							</span>
							<span role="cell" class="hidden font-semibold tabular-nums md:block"
								>#{d.deployment_number}</span
							>
							<span role="cell" class="max-md:hidden">
								<span
									class={cn(
										'rounded-sm px-1.5 py-0.5 text-[11.5px] font-semibold',
										d.environment === 'production'
											? 'bg-accent-soft text-primary'
											: 'bg-raised text-muted-foreground'
									)}>{d.environment}</span
								>
							</span>
							<span role="cell" class="min-w-0 truncate max-md:col-span-2">
								{l.translate(`deployment.source.${d.source.discriminator}`)}
								{#if d.source.discriminator === 'git'}
									· {d.source.data.branch}
									{#if d.source.data.hash}
										· <span class="font-mono">{d.source.data.hash.slice(0, 7)}</span>
									{/if}
								{/if}
								{#if d.state.error_code}
									· <span class="font-mono text-danger">{d.state.error_code}</span>
								{/if}
							</span>
							<span role="cell" class="truncate text-muted-foreground max-md:hidden"
								>{d.requested_by.email}</span
							>
							<span role="cell" class="text-muted-foreground tabular-nums max-md:col-span-2"
								>{timing(d)}</span
							>
							<span role="cell" class="max-md:hidden">
								{#if live}
									<span class="flex items-center gap-1.5 text-xs font-semibold text-success">
										<span class="size-2 rounded-full bg-current"></span>
										{l.translate('detail.card.live')}
									</span>
								{/if}
							</span>
							<span role="cell" class="relative z-10 max-md:col-start-2 max-md:row-start-1">
								<DropdownMenu.Root>
									<DropdownMenu.Trigger>
										{#snippet child({ props })}
											<Button
												{...props}
												variant="outline"
												size="icon-sm"
												class="data-[state=open]:bg-line-strong"
												aria-label={l.translate('detail.row.menu', [d.deployment_number])}
											>
												<Icon icon={MoreHorizontalIcon} />
											</Button>
										{/snippet}
									</DropdownMenu.Trigger>
									<DropdownMenu.Content align="end" sideOffset={4} class="w-[212px] p-1.5">
										<DropdownMenu.Item
											class="h-9 gap-2.5 px-2.5 whitespace-nowrap"
											onSelect={() => actions.openPreview(d.deployment_number)}
										>
											<Icon icon={CommandLineIcon} class="text-muted-foreground" />
											{l.translate('detail.row.logs')}
										</DropdownMenu.Item>
										{#if !app.cleanup_requested_at}
											{#if d.environment === 'staging' && d.state.status === DeploymentStatus.Succeeded}
												<DropdownMenu.Item
													class="h-9 gap-2.5 px-2.5 whitespace-nowrap"
													onSelect={() => {
														promoteNumber = d.deployment_number;
														promoteOpen = true;
													}}
												>
													<Icon icon={Rocket01Icon} class="text-muted-foreground" />
													{l.translate('detail.row.promote')}
												</DropdownMenu.Item>
											{/if}
											{#if d.state.finished_at}
												<DropdownMenu.Item
													class="h-9 gap-2.5 px-2.5 whitespace-nowrap"
													onSelect={() => {
														redeploy = d;
														redeployOpen = true;
													}}
												>
													<Icon icon={RepeatIcon} class="text-muted-foreground" />
													{l.translate('deployment.redeploy')}
												</DropdownMenu.Item>
											{/if}
										{/if}
										<DropdownMenu.Item
											class="h-9 gap-2.5 px-2.5 whitespace-nowrap"
											onSelect={() => copyLink(d)}
										>
											<Icon icon={Copy01Icon} class="text-muted-foreground" />
											{l.translate('detail.row.copy')}
										</DropdownMenu.Item>
									</DropdownMenu.Content>
								</DropdownMenu.Root>
							</span>
						</div>
					{/each}

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
