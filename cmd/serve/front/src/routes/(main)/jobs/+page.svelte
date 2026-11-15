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
	import CancelCircleIcon from '@hugeicons/core-free-icons/CancelCircleIcon';
	import InformationCircleIcon from '@hugeicons/core-free-icons/InformationCircleIcon';
	import Layers01Icon from '@hugeicons/core-free-icons/Layers01Icon';
	import Loading03Icon from '@hugeicons/core-free-icons/Loading03Icon';
	import TextIcon from '@hugeicons/core-free-icons/TextIcon';
	import { onMount } from 'svelte';
	import CodeBlock from '#lib/components/code-block.svelte';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import Drawer from '#lib/components/drawer.svelte';
	import EmptyState from '#lib/components/empty-state.svelte';
	import Icon from '#lib/components/icon.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import StatusBadge, { type BadgeState } from '#lib/components/status-badge.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import EnvironmentChip from '#lib/components/environment-chip.svelte';
	import ColumnsMenu from '#lib/components/table/columns-menu.svelte';
	import FilterMenu, { type FilterOption } from '#lib/components/table/filter-menu.svelte';
	import RowMenuGroup from '#lib/components/table/row-menu-group.svelte';
	import RowMenuItem from '#lib/components/table/row-menu-item.svelte';
	import RowMenu from '#lib/components/table/row-menu.svelte';
	import * as Table from '#lib/components/ui/table/index.js';
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
	import { ACTIONS_CELL, ACTIONS_HEAD, TableColumns } from '#lib/table-columns.svelte.ts';

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

	const columns = new TableColumns(
		[
			{
				id: 'state',
				label: l.translate('column.state'),
				icon: CheckmarkCircle02Icon,
				width: 'w-[14%]',
				locked: true
			},
			{
				id: 'job',
				label: l.translate('column.job'),
				icon: Task01Icon,
				width: 'w-[16%]',
				locked: true
			},
			{ id: 'subject', label: l.translate('column.subject'), icon: TextIcon, width: 'w-[15%]' },
			{
				id: 'environment',
				label: l.translate('column.environment'),
				icon: Layers01Icon,
				width: 'w-[13%]',
				priority: 2
			},
			{
				id: 'detail',
				label: l.translate('column.detail'),
				icon: InformationCircleIcon,
				width: 'w-[18%]',
				priority: 3
			},
			{
				id: 'retry',
				label: l.translate('column.retry'),
				icon: RefreshIcon,
				width: 'w-[10%]',
				priority: 3
			},
			{ id: 'queued', label: l.translate('column.queued'), icon: Clock01Icon, width: 'w-[9%]' }
		],
		'jobs'
	);

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
	<div class="flex flex-wrap items-center justify-between gap-2">
		<div class="flex flex-wrap items-center gap-3">
			<FilterMenu
				label={l.translate('column.state')}
				icon={Layers01Icon}
				value={stateFilter || 'any'}
				options={stateOptions}
				onValueChange={(v) => setStatus(v === 'any' ? '' : v)}
			/>
			<span class="text-[12.5px] text-faint">{l.translate('jobs.order')}</span>
		</div>
		<ColumnsMenu {columns} />
	</div>

	{#if jobs && jobs.data.length > 0}
		<div class="flex flex-col gap-2">
			<Table.Root aria-label={l.translate('breadcrumb.jobs')}>
				<Table.Header>
					<Table.Row>
						{#each columns.shown as column (column.id)}
							<Table.Head class={columns.head(column.id)}>{column.label}</Table.Head>
						{/each}
						<Table.Head class={ACTIONS_HEAD}>{l.translate('column.actions')}</Table.Head>
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each jobs.data as job (job.id)}
						{@const current = jobState(job, jobs.data)}
						{@const badge = badges[current.state]}
						{@const subject = jobSubject(job, names)}
						{@const openText = openLabel[subject.kind]}
						<Table.Row
							class="relative"
							data-state={drawerOpen && selectedId === job.id ? 'selected' : undefined}
						>
							{#if columns.visible('state')}
								<Table.Cell class={columns.cell('state')}>
									<StatusBadge state={badge.state} label={badge.label} />
								</Table.Cell>
							{/if}
							{#if columns.visible('job')}
								<Table.Cell class={columns.cell('job')}>
									<!-- The whole row opens the drawer; the button stretches over it. -->
									<button
										type="button"
										class="font-medium outline-none after:absolute after:inset-0 after:rounded-md focus-visible:after:ring-2 focus-visible:after:ring-ring focus-visible:after:ring-inset"
										onclick={() => open(job)}
									>
										{l.translate(job.message_name as AppTranslationsString)}
									</button>
								</Table.Cell>
							{/if}
							{#if columns.visible('subject')}
								<Table.Cell class={columns.cell('subject', 'font-semibold')} title={subject.label}>
									{subject.label}
								</Table.Cell>
							{/if}
							{#if columns.visible('environment')}
								<Table.Cell class={columns.cell('environment')}>
									{#if subject.environment}
										<EnvironmentChip environment={subject.environment} />
									{:else}
										<span class="text-faint">—</span>
									{/if}
								</Table.Cell>
							{/if}
							{#if columns.visible('detail')}
								<Table.Cell
									class={columns.cell(
										'detail',
										job.error_code
											? 'font-mono text-[12.5px] text-danger'
											: current.after
												? 'text-muted-foreground'
												: 'font-mono text-[12.5px] text-faint'
									)}
									title={job.error_code ?? job.resource_id}
								>
									{#if job.error_code}
										{job.error_code}
									{:else if current.after}
										{l.translate('jobs.after', [jobSubject(current.after, names).label])}
									{:else}
										{job.resource_id}
									{/if}
								</Table.Cell>
							{/if}
							{#if columns.visible('retry')}
								<Table.Cell class={columns.cell('retry', 'text-muted-foreground tabular-nums')}>
									{current.state === 'failed'
										? l.translate('jobs.next_in', [secondsUntil(job.not_before, now)])
										: '—'}
								</Table.Cell>
							{/if}
							{#if columns.visible('queued')}
								<Table.Cell class={columns.cell('queued', 'text-muted-foreground tabular-nums')}>
									{l.time(job.queued_at)}
								</Table.Cell>
							{/if}
							<Table.Cell class={ACTIONS_CELL}>
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
							</Table.Cell>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>

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
