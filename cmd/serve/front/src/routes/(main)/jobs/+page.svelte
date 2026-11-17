<!--
	Jobs v4 (DESIGN.md §11.9): the live queue, one row per job named for what it acts on ("welcome #5",
	staging), grouped by state rather than by day (§5 Lists and tables): a queue drains in minutes,
	and what matters is what is stuck. 10 at a time, Show more for the next. States come from the server: Running (retrieved), Failed, retrying (an error, next try
	in N s), Waiting (behind a running job of its group, when it is on the page) and Pending. Cancel
	only on cancellable jobs, behind a confirmation. Polls every 5 s; pauses on demand, while a job's
	drawer is open and while the tab is hidden. The State filter maps to GET /jobs?status=.
-->
<script lang="ts">
	import ArrowDown01Icon from '@hugeicons/core-free-icons/ArrowDown01Icon';
	import ArrowUpRight01Icon from '@hugeicons/core-free-icons/ArrowUpRight01Icon';
	import Cancel01Icon from '@hugeicons/core-free-icons/Cancel01Icon';
	import CheckmarkCircle02Icon from '@hugeicons/core-free-icons/CheckmarkCircle02Icon';
	import Clock01Icon from '@hugeicons/core-free-icons/Clock01Icon';
	import RefreshIcon from '@hugeicons/core-free-icons/RefreshIcon';
	import Task01Icon from '@hugeicons/core-free-icons/Task01Icon';
	import CancelCircleIcon from '@hugeicons/core-free-icons/CancelCircleIcon';
	import InformationCircleIcon from '@hugeicons/core-free-icons/InformationCircleIcon';
	import Layers01Icon from '@hugeicons/core-free-icons/Layers01Icon';
	import Loading03Icon from '@hugeicons/core-free-icons/Loading03Icon';
	import { onMount } from 'svelte';
	import { fromStore } from 'svelte/store';
	import type { IconData } from '#lib/components/icon.svelte';
	import TimelineGroup from '#lib/components/timeline/timeline-group.svelte';
	import TimelineRow, { type TimelineTone } from '#lib/components/timeline/timeline-row.svelte';
	import { groupBy } from '#lib/timeline.ts';
	import CodeBlock from '#lib/components/code-block.svelte';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import Drawer from '#lib/components/drawer.svelte';
	import EmptyState from '#lib/components/empty-state.svelte';
	import Icon from '#lib/components/icon.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import StatusBadge, { type BadgeState } from '#lib/components/status-badge.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import EnvironmentChip from '#lib/components/environment-chip.svelte';
	import FilterMenu, { type FilterOption } from '#lib/components/table/filter-menu.svelte';
	import RowMenuGroup from '#lib/components/table/row-menu-group.svelte';
	import RowMenuItem from '#lib/components/table/row-menu-item.svelte';
	import RowMenu from '#lib/components/table/row-menu.svelte';
	import {
		isCancellable,
		jobState,
		jobSubject,
		prettyPayload,
		secondsUntil,
		type JobState,
		type JobSubject
	} from '#lib/job.ts';
	import l, { type AppTranslationsString } from '#lib/localization/index.ts';
	import appsService from '#lib/resources/apps.ts';
	import service, { JobPolicy, type Job, type JobStatusFilter } from '#lib/resources/jobs.ts';
	import targetsService from '#lib/resources/targets.ts';

	/** How many pages of 10 are shown; Show more adds one. */
	let pages = $state(1);
	let stateFilter = $state<'' | JobStatusFilter>('');
	let paused = $state(false);
	let visible = $state(true);
	let now = $state(Date.now());

	let selectedId = $state<string>();
	let drawerOpen = $state(false);
	let cancelTarget = $state<Job>();
	let cancelOpen = $state(false);

	const live = $derived(!paused && visible && !drawerOpen);
	// One query per page shown, joined; a job that moved up a page meanwhile is kept once.
	const pageQueries = $derived(
		Array.from({ length: pages }, (_, i) =>
			fromStore(service.queryAll({ page: i + 1, status: stateFilter || undefined }, live).data)
		)
	);
	const jobs = $derived(pageQueries[0].current);
	const rows = $derived.by(() => {
		const seen: Record<string, true> = {};
		return pageQueries
			.flatMap((q) => q.current?.data ?? [])
			.filter((job) => !seen[job.id] && (seen[job.id] = true));
	});
	const groups = $derived(
		groupBy(rows, (job) => jobState(job, rows).state, ['running', 'failed', 'waiting', 'pending'])
	);
	const loadingMore = $derived(pages > 1 && !pageQueries[pages - 1].current);

	const { data: apps } = appsService.queryAll();
	const { data: targets } = targetsService.queryAll();
	const names = $derived({
		app: (id: string) => $apps?.find((app) => app.id === id)?.name,
		target: (id: string) => $targets?.find((target) => target.id === id)?.name
	});

	onMount(() => {
		const onVisibility = () => (visible = document.visibilityState === 'visible');
		document.addEventListener('visibilitychange', onVisibility);
		const clock = setInterval(() => (now = Date.now()), 1000);
		return () => {
			document.removeEventListener('visibilitychange', onVisibility);
			clearInterval(clock);
		};
	});

	// Keep showing the last known copy if the job leaves the queue while its drawer is open.
	let selected = $state<Job>();
	$effect(() => {
		const fresh = rows.find((j) => j.id === selectedId);
		if (fresh) selected = fresh;
	});

	const stateOptions: FilterOption<'any' | JobStatusFilter>[] = [
		{ value: 'any', label: l.translate('jobs.filter.any'), icon: Layers01Icon },
		{
			value: 'running',
			label: l.translate('status.running'),
			icon: Loading03Icon,
			tone: 'text-warning'
		},
		{
			value: 'failed',
			label: l.translate('jobs.failed_retrying'),
			icon: CancelCircleIcon,
			tone: 'text-danger'
		},
		{ value: 'pending', label: l.translate('status.pending'), icon: Clock01Icon }
	];

	const badges: Record<JobState, { state: BadgeState; label: string }> = {
		running: { state: 'running', label: l.translate('status.running') },
		failed: { state: 'failed', label: l.translate('jobs.failed_retrying') },
		waiting: { state: 'pending', label: l.translate('jobs.state.waiting') },
		pending: { state: 'pending', label: l.translate('status.pending') }
	};

	const policyRows = [
		{
			flag: JobPolicy.PreserveOrder,
			name: 'jobs.policy.preserve_group_order',
			help: 'jobs.policy.preserve.help'
		},
		{
			flag: JobPolicy.WaitForOthersResourceID,
			name: 'jobs.policy.wait_others_resource_id',
			help: 'jobs.policy.wait.help'
		},
		{
			flag: JobPolicy.Cancellable,
			name: 'jobs.policy.cancellable',
			help: 'jobs.policy.cancellable.help'
		},
		{ flag: JobPolicy.Mergeable, name: 'jobs.policy.mergeable', help: 'jobs.policy.mergeable.help' }
	] satisfies { flag: JobPolicy; name: AppTranslationsString; help: AppTranslationsString }[];

	const openLabel: Record<JobSubject['kind'], AppTranslationsString | undefined> = {
		deployment: 'jobs.open.deployment',
		app: 'jobs.open.app',
		target: 'jobs.open.target',
		unknown: undefined
	};

	const tiles: Record<JobState, { icon: IconData; tone: TimelineTone; spin: boolean }> = {
		running: { icon: Loading03Icon, tone: 'warning', spin: true },
		failed: { icon: CancelCircleIcon, tone: 'danger', spin: false },
		waiting: { icon: Clock01Icon, tone: 'muted', spin: false },
		pending: { icon: Clock01Icon, tone: 'muted', spin: false }
	};

	function setStatus(value: string) {
		stateFilter = value as '' | JobStatusFilter;
		pages = 1;
	}

	function open(job: Job) {
		selectedId = job.id;
		selected = job;
		drawerOpen = true;
	}

	function askCancel(job: Job) {
		cancelTarget = job;
		cancelOpen = true;
	}

	async function cancel() {
		const job = cancelTarget!;
		try {
			await service.delete(job.id);
		} finally {
			await service
				.fetchAll({ page: 1, status: stateFilter || undefined }, { cache: 'no-store' })
				.catch(() => {});
		}
		if (job.id === selectedId) drawerOpen = false;
	}
</script>

<PageHeader title={l.translate('breadcrumb.jobs')}>
	{#snippet actions()}
		<Button variant="outline" onclick={() => (paused = !paused)}>
			<Icon icon={paused ? RefreshIcon : Clock01Icon} />
			{l.translate(paused ? 'jobs.resume' : 'jobs.pause')}
		</Button>
	{/snippet}
	{#snippet status()}
		{#if jobs}
			<span>{l.translate('jobs.summary', [jobs.total])}</span>
		{/if}
		<span class="flex items-center gap-2" aria-live="polite">
			{#if live}
				<span class="size-2 rounded-full bg-success ring-3 ring-success-soft"></span>
				{l.translate('jobs.live')}
			{:else}
				<Icon icon={Clock01Icon} class="size-3.5" />
				{l.translate('jobs.paused')}
			{/if}
		</span>
		<span>{l.translate('jobs.retry_note')}</span>
	{/snippet}
</PageHeader>

<div class="flex flex-col gap-3.5">
	<div class="flex flex-wrap items-center justify-between gap-2">
		<div class="flex flex-wrap items-center gap-3">
			<FilterMenu
				label={l.translate('column.state')}
				icon={Layers01Icon}
				value={stateFilter || 'any'}
				options={stateOptions}
				onValueChange={(v) => setStatus(v === 'any' ? '' : v)}
			/>
			<span class="text-xs text-muted-foreground">{l.translate('jobs.order')}</span>
		</div>
	</div>

	{#if jobs && rows.length > 0}
		<div class="flex flex-col gap-4">
			{#each groups as group (group.key)}
				<TimelineGroup label={badges[group.key].label} count={group.items.length}>
					{#each group.items as job (job.id)}
						{@const current = jobState(job, rows)}
						{@const tile = tiles[current.state]}
						{@const subject = jobSubject(job, names)}
						{@const openText = openLabel[subject.kind]}
						{@const kind = l.translate(job.message_name as AppTranslationsString)}
						{@const detail =
							job.error_code ??
							(current.after
								? l.translate('jobs.after', [jobSubject(current.after, names).label])
								: job.resource_id)}
						<TimelineRow
							icon={tile.icon}
							tone={tile.tone}
							spin={tile.spin}
							label={[
								subject.label,
								badges[current.state].label,
								kind,
								detail,
								l.translate('jobs.queued_time', [l.time(job.queued_at)])
							].join(', ')}
							selected={drawerOpen && selectedId === job.id}
							onclick={() => open(job)}
						>
							{#snippet title()}{subject.label}{/snippet}
							{#snippet badge()}
								{#if subject.environment}<EnvironmentChip environment={subject.environment} />{/if}
							{/snippet}
							{#snippet details()}
								{kind}
								·
								<span
									class={job.error_code
										? 'font-mono text-danger'
										: current.after
											? ''
											: 'font-mono'}>{detail}</span
								>
								{#if current.state === 'failed'}
									· {l.translate('jobs.next_in', [secondsUntil(job.not_before, now)])}
								{/if}
								· {l.translate('jobs.queued_time', [l.time(job.queued_at)])}
							{/snippet}
							{#snippet actions()}
								<RowMenu label={l.translate('table.menu', [subject.label])}>
									<RowMenuGroup>
										<RowMenuItem icon={InformationCircleIcon} onSelect={() => open(job)}>
											{l.translate('jobs.details')}
										</RowMenuItem>
										{#if subject.href && openText}
											<RowMenuItem icon={ArrowUpRight01Icon} href={subject.href}>
												{l.translate(openText)}
											</RowMenuItem>
										{/if}
									</RowMenuGroup>
									{#if isCancellable(job)}
										<RowMenuGroup>
											<RowMenuItem
												icon={Cancel01Icon}
												variant="destructive"
												onSelect={() => askCancel(job)}
											>
												{l.translate('jobs.cancel')}
											</RowMenuItem>
										</RowMenuGroup>
									{/if}
								</RowMenu>
							{/snippet}
						</TimelineRow>
					{/each}
				</TimelineGroup>
			{/each}

			<div class="flex flex-wrap items-center gap-3 text-label text-muted-foreground">
				{#if rows.length < jobs.total}
					<Button variant="outline" size="sm" loading={loadingMore} onclick={() => pages++}>
						<Icon icon={ArrowDown01Icon} />
						{l.translate('list.show_more', [Math.min(10, jobs.total - rows.length)])}
					</Button>
				{/if}
				<span class="tabular-nums">{l.translate('jobs.count', [rows.length, jobs.total])}</span>
			</div>
		</div>
	{:else if jobs && stateFilter}
		<EmptyState
			icon={Task01Icon}
			title={l.translate('jobs.empty.filtered.title')}
			description={l.translate('jobs.empty.filtered.description')}
		>
			{#snippet action()}
				<Button variant="outline" onclick={() => setStatus('')}
					>{l.translate('jobs.filter.any')}</Button
				>
			{/snippet}
		</EmptyState>
	{:else if jobs}
		<EmptyState
			icon={CheckmarkCircle02Icon}
			title={l.translate('jobs.empty.title')}
			description={l.translate('jobs.empty.description')}
		/>
	{/if}
</div>

{#if selected}
	{@const job = selected}
	{@const current = jobState(job, rows)}
	{@const badge = badges[current.state]}
	{@const subject = jobSubject(job, names)}
	{@const openText = openLabel[subject.kind]}
	{@const rules = policyRows.filter((row) => job.policy & row.flag)}
	<Drawer bind:open={drawerOpen} title={l.translate(job.message_name as AppTranslationsString)}>
		{#snippet description()}{subject.label}{subject.environment
				? ` · ${subject.environment}`
				: ''}{/snippet}

		<div class="flex flex-wrap items-center gap-2.5">
			<StatusBadge state={badge.state} label={badge.label} />
			{#if current.state === 'failed'}
				<span class="text-label text-muted-foreground tabular-nums">
					{l.translate('jobs.next_in', [secondsUntil(job.not_before, now)])}
				</span>
			{:else if current.after}
				<span class="text-label text-muted-foreground">
					{l.translate('jobs.after', [jobSubject(current.after, names).label])}
				</span>
			{/if}
		</div>

		{#if job.error_code}
			<section class="flex flex-col gap-2">
				<h3 class="text-xs font-semibold tracking-wide text-muted-foreground">
					{l.translate('jobs.last_error')}
				</h3>
				<pre
					class="max-h-48 overflow-auto rounded-md bg-danger-soft px-3 py-2.5 font-mono text-xs/[18px] whitespace-pre-wrap">{job.error_code}</pre>
				<p class="text-xs text-muted-foreground">{l.translate('jobs.last_error.help')}</p>
			</section>
		{/if}

		<dl class="grid grid-cols-[110px_minmax(0,1fr)] gap-x-4 gap-y-2.5 text-label/5">
			<dt class="text-muted-foreground">{l.translate('jobs.for')}</dt>
			<dd class="min-w-0">
				{#if subject.href}
					<a href={subject.href} class="text-primary hover:underline">{subject.label}</a>
				{:else}
					{subject.label}
				{/if}
				{#if subject.environment}
					<span class="text-muted-foreground"> · {subject.environment}</span>
				{/if}
			</dd>
			<dt class="text-muted-foreground">{l.translate('jobs.resource.label')}</dt>
			<dd class="font-mono break-all text-muted-foreground">{job.resource_id}</dd>
			<dt class="text-muted-foreground">{l.translate('jobs.group.label')}</dt>
			<dd class="flex min-w-0 flex-col gap-0.5">
				<span class="font-mono break-all">{job.group}</span>
				<span class="text-muted-foreground">{l.translate('jobs.group.help')}</span>
			</dd>
			<dt class="text-muted-foreground">{l.translate('jobs.queued')}</dt>
			<dd class="tabular-nums">{l.datetime(job.queued_at)}</dd>
			<dt class="text-muted-foreground">{l.translate('jobs.not_before.label')}</dt>
			<dd class="tabular-nums">{l.datetime(job.not_before)}</dd>
		</dl>

		<section class="flex flex-col gap-2">
			<h3 class="text-xs font-semibold tracking-wide text-muted-foreground">
				{l.translate('jobs.policy.title')}
			</h3>
			{#each rules as rule (rule.flag)}
				<p class="grid grid-cols-[16px_minmax(0,1fr)] gap-2 text-label/[19px]">
					<Icon icon={CheckmarkCircle02Icon} class="mt-0.5 size-4 text-success" />
					<span>
						<strong class="font-semibold">{l.translate(rule.name)}</strong>
						<span class="text-muted-foreground"> · {l.translate(rule.help)}</span>
					</span>
				</p>
			{:else}
				<p class="text-label text-muted-foreground">{l.translate('jobs.policy.none')}</p>
			{/each}
		</section>

		<section class="flex flex-col gap-2">
			<h3 class="text-xs font-semibold tracking-wide text-muted-foreground">
				{l.translate('jobs.payload.title')}
			</h3>
			<CodeBlock
				code={prettyPayload(job.message_data)}
				lang="json"
				copy
				label={l.translate('jobs.payload.title')}
			/>
		</section>

		{#snippet footer()}
			{#if subject.href && openText}
				<Button variant="outline" href={subject.href} class="mr-auto">
					<Icon icon={ArrowUpRight01Icon} />
					{l.translate(openText)}
				</Button>
			{/if}
			{#if isCancellable(job)}
				<Button variant="destructive" onclick={() => askCancel(job)}>
					<Icon icon={Cancel01Icon} />
					{l.translate('jobs.cancel')}
				</Button>
			{:else}
				<Button variant="outline" onclick={() => (drawerOpen = false)}
					>{l.translate('close')}</Button
				>
			{/if}
		{/snippet}
	</Drawer>
{/if}

<ConfirmDialog
	bind:open={cancelOpen}
	destructive
	icon={Cancel01Icon}
	title={l.translate('jobs.cancel.title')}
	description={cancelTarget
		? l.translate('jobs.cancel.subject', [jobSubject(cancelTarget, names).label])
		: l.translate('jobs.cancel.description')}
	confirmLabel={l.translate('jobs.cancel')}
	onconfirm={cancel}
/>
