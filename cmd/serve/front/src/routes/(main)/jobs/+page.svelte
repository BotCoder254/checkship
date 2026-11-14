<!--
	Jobs (DESIGN.md §5, §11.6): one row per queued job on a sunken fill; a row opens its details in
	a drawer (group, policy, error, payload highlighted as JSON). Cancel stays on cancellable rows
	and is repeated in the drawer, both behind a confirmation. Polls every 5 seconds.
-->
<script lang="ts">
	import ArrowLeft01Icon from '@hugeicons/core-free-icons/ArrowLeft01Icon';
	import ArrowRight01Icon from '@hugeicons/core-free-icons/ArrowRight01Icon';
	import Cancel01Icon from '@hugeicons/core-free-icons/Cancel01Icon';
	import Task01Icon from '@hugeicons/core-free-icons/Task01Icon';
	import CodeBlock from '#lib/components/code-block.svelte';
	import ConfirmDialog from '#lib/components/confirm-dialog.svelte';
	import Drawer from '#lib/components/drawer.svelte';
	import EmptyState from '#lib/components/empty-state.svelte';
	import Icon from '#lib/components/icon.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import StatusBadge, { type BadgeState } from '#lib/components/status-badge.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import l, { type AppTranslationsString } from '#lib/localization/index.ts';
	import service, { JobPolicy, type Job } from '#lib/resources/jobs.ts';
	import { cn } from '#lib/utils.js';

	let page = $state(1);
	const query = $derived(service.queryAll(page));
	const store = $derived(query.data);
	const jobs = $derived($store);

	let selectedId = $state<string>();
	let drawerOpen = $state(false);
	let cancelId = $state<string>();
	let cancelOpen = $state(false);

	// Keep showing the last known copy if the job leaves the queue while its drawer is open.
	let selected = $state<Job>();
	$effect(() => {
		const fresh = jobs?.data.find((j) => j.id === selectedId);
		if (fresh) selected = fresh;
	});

	function jobBadge(job: Job): { state: BadgeState; label: string } {
		if (job.retrieved) return { state: 'running', label: l.translate('status.running') };
		if (job.error_code) return { state: 'failed', label: l.translate('jobs.failed_retrying') };
		return { state: 'pending', label: l.translate('status.pending') };
	}

	const cancellable = (job: Job) => (job.policy & JobPolicy.Cancellable) !== 0;

	function policies(policy: number) {
		const names: AppTranslationsString[] = [];
		if (policy & JobPolicy.PreserveOrder) names.push('jobs.policy.preserve_group_order');
		if (policy & JobPolicy.WaitForOthersResourceID)
			names.push('jobs.policy.wait_others_resource_id');
		if (policy & JobPolicy.Cancellable) names.push('jobs.policy.cancellable');
		if (policy & JobPolicy.Mergeable) names.push('jobs.policy.mergeable');
		return names.map((n) => l.translate(n));
	}

	function pretty(data: string) {
		try {
			return JSON.stringify(JSON.parse(data), null, 2);
		} catch {
			return data;
		}
	}

	function open(job: Job) {
		selectedId = job.id;
		selected = job;
		drawerOpen = true;
	}

	function askCancel(job: Job) {
		cancelId = job.id;
		cancelOpen = true;
	}

	async function cancel() {
		await service.delete(cancelId!);
		await service.fetchAll(page, { cache: 'no-store' });
		if (cancelId === selectedId) drawerOpen = false;
	}
</script>

<PageHeader title={l.translate('breadcrumb.jobs')}>
	{#snippet status()}
		{#if jobs}
			<span>{l.translate('jobs.summary', [jobs.total])}</span>
		{/if}
	{/snippet}
</PageHeader>

{#if jobs && jobs.data.length > 0}
	<div role="table" aria-label={l.translate('breadcrumb.jobs')} class="flex flex-col gap-1.5">
		<div
			role="row"
			class="hidden grid-cols-[140px_minmax(0,1fr)_200px_120px_16px] gap-4 px-4 pb-1.5 text-xs font-semibold text-muted-foreground md:grid"
		>
			<span role="columnheader">{l.translate('jobs.status')}</span>
			<span role="columnheader">{l.translate('jobs.resource')}</span>
			<span role="columnheader">{l.translate('jobs.dates')}</span>
			<span role="columnheader"><span class="sr-only">{l.translate('jobs.cancel')}</span></span>
			<span role="columnheader"></span>
		</div>
		{#each jobs.data as job (job.id)}
			{@const badge = jobBadge(job)}
			<div
				role="row"
				class={cn(
					'relative grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-4 gap-y-2 rounded-md bg-sunken py-3 pr-3 pl-4 transition-colors hover:bg-hover md:grid-cols-[140px_minmax(0,1fr)_200px_120px_16px] md:py-2',
					drawerOpen && selectedId === job.id && 'bg-raised'
				)}
			>
				<span role="cell" class="max-md:col-span-2">
					<StatusBadge state={badge.state} label={badge.label} />
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
					<span class="truncate font-mono text-xs text-muted-foreground">{job.resource_id}</span>
				</span>
				<span role="cell" class="flex flex-col text-[13px]/[18px] tabular-nums max-md:col-span-2">
					<span>{l.translate('jobs.queued_at', [l.datetime(job.queued_at)])}</span>
					<span class="text-muted-foreground"
						>{l.translate('jobs.not_before', [l.datetime(job.not_before)])}</span
					>
				</span>
				<span
					role="cell"
					class="relative z-10 flex justify-end max-md:col-start-2 max-md:row-start-2"
				>
					{#if cancellable(job)}
						<Button variant="outline" size="sm" onclick={() => askCancel(job)}>
							<Icon icon={Cancel01Icon} />
							{l.translate('jobs.cancel')}
						</Button>
					{/if}
				</span>
				<span role="cell" class="hidden md:block">
					<Icon icon={ArrowRight01Icon} class="size-4 text-faint" />
				</span>
			</div>
		{/each}

		{#if !(jobs.first_page && jobs.last_page)}
			<div class="flex items-center justify-between pt-2.5 text-[13px] text-muted-foreground">
				<span
					>{l.translate('app.history.page', [
						jobs.page,
						Math.max(1, Math.ceil(jobs.total / jobs.per_page))
					])}</span
				>
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
			</div>
		{/if}
	</div>
{:else if jobs}
	<EmptyState
		icon={Task01Icon}
		title={l.translate('jobs.empty.title')}
		description={l.translate('jobs.empty.description')}
	/>
{/if}

{#if selected}
	{@const job = selected}
	{@const badge = jobBadge(job)}
	<Drawer bind:open={drawerOpen} title={l.translate(job.message_name as AppTranslationsString)}>
		{#snippet description()}<span class="font-mono text-[13px]">{job.resource_id}</span>{/snippet}

		<div class="flex flex-wrap items-center gap-2.5">
			<StatusBadge state={badge.state} label={badge.label} />
			{#if job.error_code}
				<span class="text-[13px] text-muted-foreground">
					{l.translate('jobs.next_try', [l.time(job.not_before)])}
				</span>
			{/if}
		</div>

		<dl
			class="grid grid-cols-[100px_minmax(0,1fr)] gap-x-4 gap-y-2.5 rounded-md bg-sunken p-4 text-[13px]/5 [&_dt]:first-letter:uppercase"
		>
			<dt class="text-muted-foreground">{l.translate('jobs.group')}</dt>
			<dd class="font-mono break-all">{job.group}</dd>
			<dt class="text-muted-foreground">{l.translate('jobs.policy')}</dt>
			<dd class="flex flex-wrap gap-1.5">
				{#each policies(job.policy) as name (name)}
					<span class="rounded-sm bg-raised px-2 py-0.5 text-xs font-medium">{name}</span>
				{:else}
					<span class="text-muted-foreground">—</span>
				{/each}
			</dd>
			<dt class="text-muted-foreground">{l.translate('jobs.queued')}</dt>
			<dd>{l.datetime(job.queued_at)}</dd>
			<dt class="text-muted-foreground">{l.translate('jobs.not_before.label')}</dt>
			<dd>{l.datetime(job.not_before)}</dd>
			{#if job.error_code}
				<dt class="text-muted-foreground">{l.translate('jobs.error')}</dt>
				<dd class="font-mono break-all text-danger">{job.error_code}</dd>
			{/if}
		</dl>

		<section class="flex flex-col gap-2.5">
			<h3 class="text-sm font-semibold">{l.translate('jobs.payload.title')}</h3>
			<CodeBlock
				code={pretty(job.message_data)}
				lang="json"
				copy
				label={l.translate('jobs.payload.title')}
			/>
		</section>

		{#snippet footer()}
			{#if cancellable(job)}
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
	description={l.translate('jobs.cancel.description')}
	confirmLabel={l.translate('jobs.cancel')}
	onconfirm={cancel}
/>
