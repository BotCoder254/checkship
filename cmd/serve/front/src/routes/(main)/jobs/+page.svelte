<!--
	Jobs v4 (DESIGN.md §11.9): the live queue, one row per job named for what it acts on ("welcome #5",
	staging). States come from the server: Running (retrieved), Failed, retrying (an error, next try
	in N s), Waiting (behind a running job of its group, when it is on the page) and Pending. Cancel
	only on cancellable jobs, behind a confirmation. Polls every 5 s; pauses on demand, while a job's
	drawer is open and while the tab is hidden. The State filter maps to GET /jobs?status=.
-->
<script lang="ts">
	import ArrowLeft01Icon from '@hugeicons/core-free-icons/ArrowLeft01Icon';
	import ArrowRight01Icon from '@hugeicons/core-free-icons/ArrowRight01Icon';
	import ArrowUpRight01Icon from '@hugeicons/core-free-icons/ArrowUpRight01Icon';
	import Cancel01Icon from '@hugeicons/core-free-icons/Cancel01Icon';
	import CheckmarkCircle02Icon from '@hugeicons/core-free-icons/CheckmarkCircle02Icon';
	import Clock01Icon from '@hugeicons/core-free-icons/Clock01Icon';
	import RefreshIcon from '@hugeicons/core-free-icons/RefreshIcon';
	import Task01Icon from '@hugeicons/core-free-icons/Task01Icon';
	import { onMount } from 'svelte';
	import CodeBlock from '#lib/components/code-block.svelte';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import Drawer from '#lib/components/drawer.svelte';
	import EmptyState from '#lib/components/empty-state.svelte';
	import Icon from '#lib/components/icon.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import StatusBadge, { type BadgeState } from '#lib/components/status-badge.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { Label } from '#lib/components/ui/label/index.js';
	import * as Select from '#lib/components/ui/select/index.js';
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
	import { cn } from '#lib/utils.js';

	let page = $state(1);
	let stateFilter = $state<'' | JobStatusFilter>('');
	let paused = $state(false);
	let visible = $state(true);
	let now = $state(Date.now());

	let selectedId = $state<string>();
	let drawerOpen = $state(false);
	let cancelTarget = $state<Job>();
	let cancelOpen = $state(false);

	const live = $derived(!paused && visible && !drawerOpen);
	const filters = $derived({ page, status: stateFilter || undefined });
	const query = $derived(service.queryAll(filters, live));
	const store = $derived(query.data);
	const jobs = $derived($store);

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
		const fresh = jobs?.data.find((j) => j.id === selectedId);
		if (fresh) selected = fresh;
	});

	const stateOptions: { value: '' | JobStatusFilter; label: string }[] = [
		{ value: '', label: l.translate('jobs.filter.any') },
		{ value: 'running', label: l.translate('status.running') },
		{ value: 'failed', label: l.translate('jobs.failed_retrying') },
		{ value: 'pending', label: l.translate('status.pending') }
	];
	const stateLabel = $derived(stateOptions.find((o) => o.value === stateFilter)?.label);

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

	const pages = $derived(jobs ? Math.max(1, Math.ceil(jobs.total / jobs.per_page)) : 1);

	function setStatus(value: string) {
		stateFilter = value as '' | JobStatusFilter;
		page = 1;
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
			await service.fetchAll(filters, { cache: 'no-store' }).catch(() => {});
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
	<div class="flex flex-wrap items-end justify-between gap-3">
		<div class="flex w-full flex-col gap-1 sm:w-55">
			<Label for="jobs-state" class="text-xs font-semibold text-muted-foreground">
				{l.translate('jobs.filter.state')}
			</Label>
			<Select.Root type="single" value={stateFilter} onValueChange={setStatus}>
				<Select.Trigger id="jobs-state" class="w-full">{stateLabel}</Select.Trigger>
				<Select.Content>
					{#each stateOptions as option (option.value)}
						<Select.Item value={option.value} label={option.label}>{option.label}</Select.Item>
					{/each}
				</Select.Content>
			</Select.Root>
		</div>
		<span class="text-[12.5px] text-faint">{l.translate('jobs.order')}</span>
	</div>

	{#if jobs && jobs.data.length > 0}
		{@const columns = 'md:grid-cols-[150px_minmax(0,1.2fr)_minmax(0,1fr)_130px_120px_32px]'}
		<div role="table" aria-label={l.translate('breadcrumb.jobs')} class="flex flex-col gap-1.5">
			<div
				role="row"
				class={cn(
					'hidden gap-4 pr-2 pb-0.5 pl-4 text-xs font-semibold text-muted-foreground md:grid',
					columns
				)}
			>
				<span role="columnheader">{l.translate('jobs.filter.state')}</span>
				<span role="columnheader">{l.translate('jobs.column.job')}</span>
				<span role="columnheader">{l.translate('jobs.column.detail')}</span>
				<span role="columnheader">{l.translate('jobs.queued')}</span>
				<span role="columnheader"><span class="sr-only">{l.translate('jobs.cancel')}</span></span>
				<span role="columnheader"><span class="sr-only">{l.translate('jobs.details')}</span></span>
			</div>

			{#each jobs.data as job (job.id)}
				{@const current = jobState(job, jobs.data)}
				{@const badge = badges[current.state]}
				{@const subject = jobSubject(job, names)}
				<div
					role="row"
					class={cn(
						'relative grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-4 gap-y-2 rounded-md bg-sunken py-3 pr-2 pl-4 transition-colors hover:bg-hover md:min-h-15 md:py-2.5',
						columns,
						drawerOpen && selectedId === job.id && 'bg-raised'
					)}
				>
					<span role="cell" class="flex min-w-0 flex-col items-start gap-1 max-md:col-span-2">
						<StatusBadge state={badge.state} label={badge.label} />
						{#if current.state === 'failed'}
							<span class="text-[11.5px] text-muted-foreground tabular-nums">
								{l.translate('jobs.next_in', [secondsUntil(job.not_before, now)])}
							</span>
						{:else if current.after}
							<span class="max-w-full truncate text-[11.5px] text-muted-foreground">
								{l.translate('jobs.after', [jobSubject(current.after, names).label])}
							</span>
						{/if}
					</span>
					<span role="cell" class="flex min-w-0 flex-col">
						<!-- The whole row opens the drawer; the button stretches over it. -->
						<button
							type="button"
							class="truncate text-left text-sm font-medium outline-none after:absolute after:inset-0 after:rounded-md focus-visible:after:ring-2 focus-visible:after:ring-ring focus-visible:after:ring-inset"
							onclick={() => open(job)}
						>
							{l.translate(job.message_name as AppTranslationsString)}
						</button>
						<span class="flex min-w-0 items-center gap-1.5">
							<span class="truncate text-[12.5px] text-muted-foreground">{subject.label}</span>
							{#if subject.environment}
								<span
									class={cn(
										'shrink-0 rounded-sm px-1.5 text-[11.5px]/5 font-semibold',
										subject.environment === 'production'
											? 'bg-accent-soft text-primary'
											: 'bg-raised text-muted-foreground'
									)}>{subject.environment}</span
								>
							{/if}
						</span>
					</span>
					<span
						role="cell"
						title={job.error_code ?? job.resource_id}
						class={cn(
							'truncate font-mono text-[12.5px] max-md:col-span-2',
							job.error_code ? 'text-danger' : 'text-faint'
						)}
					>
						{job.error_code ?? job.resource_id}
					</span>
					<span role="cell" class="truncate text-[13px] text-muted-foreground tabular-nums">
						{l.translate('jobs.queued_time', [l.time(job.queued_at)])}
					</span>
					<span role="cell" class="relative z-10 flex justify-end">
						{#if isCancellable(job)}
							<Button variant="outline" size="sm" onclick={() => askCancel(job)}>
								<Icon icon={Cancel01Icon} />
								{l.translate('jobs.cancel')}
							</Button>
						{/if}
					</span>
					<span role="cell" class="hidden md:flex md:justify-center">
						<Icon icon={ArrowRight01Icon} class="size-4 text-faint" />
					</span>
				</div>
			{/each}

			<div class="flex items-center justify-between pt-2 text-[13px] text-muted-foreground">
				<span>{l.translate('jobs.pager', [jobs.page, pages])}</span>
				{#if !(jobs.first_page && jobs.last_page)}
					<span class="flex gap-1">
						<Button
							variant="outline"
							size="icon-sm"
							aria-label={l.translate('pagination.previous')}
							disabled={jobs.first_page}
							onclick={() => page--}
						>
							<Icon icon={ArrowLeft01Icon} />
						</Button>
						<Button
							variant="outline"
							size="icon-sm"
							aria-label={l.translate('pagination.next')}
							disabled={jobs.last_page}
							onclick={() => page++}
						>
							<Icon icon={ArrowRight01Icon} />
						</Button>
					</span>
				{/if}
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
	{@const current = jobState(job, jobs?.data ?? [])}
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
				<span class="text-[13px] text-muted-foreground tabular-nums">
					{l.translate('jobs.next_in', [secondsUntil(job.not_before, now)])}
				</span>
			{:else if current.after}
				<span class="text-[13px] text-muted-foreground">
					{l.translate('jobs.after', [jobSubject(current.after, names).label])}
				</span>
			{/if}
		</div>

		{#if job.error_code}
			<section class="flex flex-col gap-2">
				<h3 class="text-sm font-semibold">{l.translate('jobs.last_error')}</h3>
				<pre
					class="max-h-48 overflow-auto rounded-md bg-danger-soft px-3 py-2.5 font-mono text-xs/[18px] whitespace-pre-wrap">{job.error_code}</pre>
				<p class="text-[12.5px] text-muted-foreground">{l.translate('jobs.last_error.help')}</p>
			</section>
		{/if}

		<dl
			class="grid grid-cols-[110px_minmax(0,1fr)] gap-x-4 gap-y-2.5 rounded-md bg-sunken p-4 text-[13px]/5"
		>
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
			<h3 class="text-sm font-semibold">{l.translate('jobs.policy.title')}</h3>
			{#each rules as rule (rule.flag)}
				<p class="grid grid-cols-[16px_minmax(0,1fr)] gap-2 text-[13px]/[19px]">
					<Icon icon={CheckmarkCircle02Icon} class="mt-0.5 size-4 text-success" />
					<span>
						<strong class="font-semibold">{l.translate(rule.name)}</strong>
						<span class="text-muted-foreground"> · {l.translate(rule.help)}</span>
					</span>
				</p>
			{:else}
				<p class="text-[13px] text-muted-foreground">{l.translate('jobs.policy.none')}</p>
			{/each}
		</section>

		<section class="flex flex-col gap-2">
			<h3 class="text-sm font-semibold">{l.translate('jobs.payload.title')}</h3>
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
