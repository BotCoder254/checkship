<!--
	Overview (DESIGN.md §11.10): a bento grid. A stats strip compares the period with the previous
	one; the deployments chart is the hero; then what needs attention, infrastructure, the live
	activity feed, applications, durations, environments and failures.

	Live: one EventSource (GET /api/v1/events) sends hints; each hint (debounced) refetches what the
	page shows. The page also refreshes every 30 s, since some server changes send no hint.
-->
<script lang="ts">
	import Add01Icon from '@hugeicons/core-free-icons/Add01Icon';
	import ArrowRight01Icon from '@hugeicons/core-free-icons/ArrowRight01Icon';
	import CancelCircleIcon from '@hugeicons/core-free-icons/CancelCircleIcon';
	import ChartHistogramIcon from '@hugeicons/core-free-icons/ChartHistogramIcon';
	import PieChartIcon from '@hugeicons/core-free-icons/PieChartIcon';
	import Tick02Icon from '@hugeicons/core-free-icons/Tick02Icon';
	import Timer02Icon from '@hugeicons/core-free-icons/Timer02Icon';
	import WifiDisconnected01Icon from '@hugeicons/core-free-icons/WifiDisconnected01Icon';
	import { onMount, untrack } from 'svelte';
	import ActivityFeed from '#lib/components/dashboard/activity-feed.svelte';
	import AppsTile from '#lib/components/dashboard/apps-tile.svelte';
	import AttentionTile from '#lib/components/dashboard/attention-tile.svelte';
	import DeploysChart from '#lib/components/dashboard/deploys-chart.svelte';
	import DurationChart from '#lib/components/dashboard/duration-chart.svelte';
	import FailuresDrawer from '#lib/components/dashboard/failures-drawer.svelte';
	import InfraTile from '#lib/components/dashboard/infra-tile.svelte';
	import ResultsDonut from '#lib/components/dashboard/results-donut.svelte';
	import StatsStrip from '#lib/components/dashboard/stats-strip.svelte';
	import Tile from '#lib/components/dashboard/tile.svelte';
	import Icon from '#lib/components/icon.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
	import { connectLive, liveChanges, liveStatus } from '#lib/live.ts';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import appsService from '#lib/resources/apps.ts';
	import dashboard, {
		DASHBOARD_FALLBACK_POLL_MS,
		type DashboardFilters,
		type DashboardRange
	} from '#lib/resources/dashboard.ts';
	import type { Environment } from '#lib/resources/deployments.ts';
	import jobsService from '#lib/resources/jobs.ts';
	import targetsService from '#lib/resources/targets.ts';

	let range = $state<DashboardRange>(30);
	let environment = $state<'' | Environment>('');
	const filters = $derived<DashboardFilters>({ range, environment: environment || undefined });

	const summaryQuery = $derived(dashboard.querySummary(filters));
	const summaryData = $derived(summaryQuery.data);
	const summaryError = $derived(summaryQuery.error);
	const summary = $derived($summaryData);

	const { data: apps } = appsService.queryAll();
	const { data: targets } = targetsService.queryAll();
	const { data: runningJobs } = jobsService.queryAll({ page: 1, status: 'running' }, false);
	const { data: pendingJobs } = jobsService.queryAll({ page: 1, status: 'pending' }, false);
	const { data: failedJobs } = jobsService.queryAll({ page: 1, status: 'failed' }, false);

	/** Incremented on every live hint and every fallback tick; tiles refetch on change. */
	let refresh = $state(0);
	let failuresOpen = $state(false);

	const appName = (id: string) => $apps?.find((a) => a.id === id)?.name;

	async function refetch() {
		await Promise.allSettled([
			dashboard.fetchSummary(filters, { cache: 'no-store' }),
			appsService.fetchAll({ cache: 'no-store' }),
			targetsService.fetchAll(undefined, { cache: 'no-store' }),
			jobsService.fetchAll({ page: 1, status: 'running' }, { cache: 'no-store' }),
			jobsService.fetchAll({ page: 1, status: 'pending' }, { cache: 'no-store' }),
			jobsService.fetchAll({ page: 1, status: 'failed' }, { cache: 'no-store' })
		]);
	}

	onMount(() => {
		const disconnect = connectLive();
		const timer = setInterval(() => {
			refresh++;
			refetch();
		}, DASHBOARD_FALLBACK_POLL_MS);
		return () => {
			disconnect();
			clearInterval(timer);
		};
	});

	$effect(() => {
		if ($liveChanges.seq === 0) return;
		untrack(() => {
			refresh++;
			refetch();
		});
	});

	const ranges: { value: DashboardRange; label: string }[] = [
		{ value: 7, label: l.translate('dashboard.range', [7]) },
		{ value: 30, label: l.translate('dashboard.range', [30]) },
		{ value: 90, label: l.translate('dashboard.range', [90]) }
	];
	const environments: { value: '' | Environment; label: string }[] = [
		{ value: '', label: l.translate('dashboard.environment.all') },
		{ value: 'production', label: l.translate('deploy.environment.production') },
		{ value: 'staging', label: l.translate('deploy.environment.staging') }
	];

	const firstRun = $derived($apps !== undefined && $apps.length === 0);
	const activeTargets = $derived(($targets ?? []).filter((t) => !t.cleanup_requested_at).length);

	const series = $derived(summary?.series ?? []);
	const kpiLoading = $derived(!summary && !$summaryError);
	const retrySummary = () => dashboard.fetchSummary(filters, { cache: 'no-store' }).catch(() => {});
</script>

<PageHeader title={l.translate('breadcrumb.overview')}>
	{#snippet actions()}
		<span
			class="mr-1 flex items-center gap-2 text-[13px] {$liveStatus === 'offline'
				? 'text-warning'
				: 'text-muted-foreground'}"
			aria-live="polite"
		>
			{#if $liveStatus === 'live'}
				<span class="size-2 rounded-full bg-success ring-3 ring-success-soft"></span>
				{l.translate('dashboard.live')}
			{:else if $liveStatus === 'connecting'}
				<span class="size-2 rounded-full bg-faint"></span>
				{l.translate('dashboard.connecting')}
			{:else}
				<Icon icon={WifiDisconnected01Icon} class="size-3.5" />
				{l.translate('dashboard.offline')}
			{/if}
		</span>
		<Select.Root
			type="single"
			value={String(range)}
			onValueChange={(value) => (range = Number(value) as DashboardRange)}
		>
			<Select.Trigger class="w-38" aria-label={l.translate('dashboard.range.label')}>
				{ranges.find((r) => r.value === range)?.label}
			</Select.Trigger>
			<Select.Content>
				{#each ranges as r (r.value)}
					<Select.Item value={String(r.value)} label={r.label}>{r.label}</Select.Item>
				{/each}
			</Select.Content>
		</Select.Root>
		<Select.Root
			type="single"
			value={environment}
			onValueChange={(value) => (environment = value as '' | Environment)}
		>
			<Select.Trigger class="w-44" aria-label={l.translate('dashboard.environment.label')}>
				{environments.find((e) => e.value === environment)?.label}
			</Select.Trigger>
			<Select.Content>
				{#each environments as e (e.value)}
					<Select.Item value={e.value} label={e.label}>{e.label}</Select.Item>
				{/each}
			</Select.Content>
		</Select.Root>
		<Button href={routes.createApp}>
			<Icon icon={Add01Icon} />
			{l.translate('app.new')}
		</Button>
	{/snippet}
	{#snippet status()}
		{#if $apps}<span>{l.translate('dashboard.status.apps', [$apps.length])}</span>{/if}
		{#if $targets}<span>{l.translate('dashboard.status.targets', [activeTargets])}</span>{/if}
		{#if summary}
			<span
				>{l.translate('dashboard.status.deployments', [summary.current.deployments, range])}</span
			>
		{/if}
	{/snippet}
</PageHeader>

{#if firstRun}
	<section
		aria-labelledby="first-run"
		class="flex max-w-2xl flex-col gap-3 rounded-lg bg-sunken p-6"
	>
		<h2 id="first-run" class="text-base font-semibold">{l.translate('dashboard.first.title')}</h2>
		<p class="text-sm text-muted-foreground">{l.translate('dashboard.first.description')}</p>
		{#each [{ done: activeTargets > 0, label: l.translate('dashboard.first.target'), href: routes.createTarget }, { done: false, label: l.translate('dashboard.first.app'), href: routes.createApp }, { done: false, label: l.translate('dashboard.first.deploy') }] as step, i (i)}
			<div
				class="grid grid-cols-[28px_minmax(0,1fr)_auto] items-center gap-3 rounded-md bg-surface p-3"
			>
				<span
					class="flex size-7 items-center justify-center rounded-full text-[13px] font-semibold {step.done
						? 'bg-success-soft text-success'
						: 'bg-raised text-muted-foreground'}"
				>
					{#if step.done}<Icon icon={Tick02Icon} class="size-3.5" />{:else}{i + 1}{/if}
				</span>
				<span class="text-sm {step.done ? 'text-muted-foreground line-through' : ''}"
					>{step.label}</span
				>
				{#if step.href && !step.done && (i === 0 || activeTargets > 0)}
					<Button size="sm" href={step.href}>
						<Icon icon={ArrowRight01Icon} />
						{l.translate('dashboard.first.go')}
					</Button>
				{/if}
			</div>
		{/each}
	</section>
{:else}
	<div class="grid grid-flow-row-dense grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
		<StatsStrip
			{summary}
			{range}
			error={$summaryError}
			onretry={retrySummary}
			class="md:col-span-2 xl:col-span-4"
		/>

		<Tile
			title={l.translate('dashboard.deploys.title')}
			icon={ChartHistogramIcon}
			class="md:col-span-2 xl:row-span-2"
			loading={kpiLoading}
			error={summary ? undefined : $summaryError}
			onretry={retrySummary}
		>
			{#snippet actions()}
				<ul class="flex flex-wrap gap-x-4 gap-y-1 text-[12.5px] text-muted-foreground">
					<li class="flex items-center gap-1.5">
						<span class="size-2 rounded-full bg-success"></span>{l.translate(
							'dashboard.series.succeeded'
						)}
					</li>
					<li class="flex items-center gap-1.5">
						<span class="size-2 rounded-full bg-danger"></span>{l.translate(
							'dashboard.series.failed'
						)}
					</li>
					<li class="flex items-center gap-1.5">
						<span class="size-2 rounded-full bg-faint"></span>{l.translate(
							'dashboard.series.other'
						)}
					</li>
				</ul>
			{/snippet}
			{#if summary && summary.current.deployments > 0}
				<DeploysChart days={series} onselect={(day) => day.failed && (failuresOpen = true)} />
				<p class="text-[12.5px] text-muted-foreground">{l.translate('dashboard.deploys.hint')}</p>
			{:else}
				<div class="flex flex-1 flex-col items-center justify-center gap-2.5 py-10 text-center">
					<Icon icon={ChartHistogramIcon} class="size-6 text-faint" />
					<strong class="text-sm">{l.translate('dashboard.empty.range.title', [range])}</strong>
					{#if range !== 90}
						<Button variant="outline" size="sm" onclick={() => (range = range === 7 ? 30 : 90)}>
							{l.translate('dashboard.range', [range === 7 ? 30 : 90])}
						</Button>
					{/if}
				</div>
			{/if}
		</Tile>

		<AttentionTile
			apps={$apps}
			targets={$targets}
			failingJobs={$failedJobs?.total}
			class="md:col-span-2"
		/>
		<InfraTile
			targets={$targets}
			jobs={{
				running: $runningJobs?.total,
				pending: $pendingJobs?.total,
				failed: $failedJobs?.total
			}}
			class="md:col-span-2"
		/>

		<ActivityFeed
			environment={filters.environment}
			{refresh}
			live={$liveStatus === 'live'}
			{appName}
			class="md:col-span-2 xl:row-span-2"
		/>
		<AppsTile {filters} {refresh} class="md:col-span-2 xl:row-span-2" />

		<Tile
			title={l.translate('dashboard.duration')}
			icon={Timer02Icon}
			class="md:col-span-2"
			loading={kpiLoading}
			error={summary ? undefined : $summaryError}
			onretry={retrySummary}
		>
			{#snippet actions()}
				<ul class="flex gap-4 text-[12.5px] text-muted-foreground">
					<li class="flex items-center gap-1.5">
						<span class="size-2 rounded-full bg-primary"></span>{l.translate(
							'dashboard.duration.median'
						)}
					</li>
					<li class="flex items-center gap-1.5">
						<span class="size-2 rounded-full bg-warning"></span>{l.translate(
							'dashboard.duration.slowest'
						)}
					</li>
				</ul>
			{/snippet}
			<DurationChart days={series} />
		</Tile>

		<Tile
			title={l.translate('dashboard.environments')}
			icon={PieChartIcon}
			loading={kpiLoading}
			error={summary ? undefined : $summaryError}
			onretry={retrySummary}
		>
			<ResultsDonut stats={summary?.by_environment ?? []} />
		</Tile>

		<Tile
			title={l.translate('dashboard.failures')}
			icon={CancelCircleIcon}
			loading={kpiLoading}
			error={summary ? undefined : $summaryError}
			onretry={retrySummary}
		>
			<div class="flex items-baseline gap-2">
				<strong class="text-[30px]/9 font-semibold tabular-nums"
					>{summary?.current.failed ?? 0}</strong
				>
				<span class="text-[13px] text-muted-foreground"
					>{l.translate('dashboard.failures.in', [range])}</span
				>
			</div>
			{#if summary?.errors.length}
				<ul class="flex flex-col gap-1.5 text-[12.5px] text-muted-foreground">
					{#each summary.errors as e (e.code)}
						<li class="flex justify-between gap-3">
							<span class="truncate font-mono">{e.code}</span>
							<b class="font-semibold text-ink tabular-nums">{e.count}</b>
						</li>
					{/each}
				</ul>
			{/if}
			<Button
				variant="outline"
				size="sm"
				class="mt-auto w-full"
				disabled={!summary?.current.failed}
				onclick={() => (failuresOpen = true)}
			>
				<Icon icon={ArrowRight01Icon} />
				{l.translate('dashboard.failures.see')}
			</Button>
		</Tile>
	</div>
{/if}

<FailuresDrawer bind:open={failuresOpen} {filters} {appName} />
