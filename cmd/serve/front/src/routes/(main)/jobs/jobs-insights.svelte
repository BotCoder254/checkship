<!--
	The queue now (DESIGN.md §11.9 Jobs): Running, Waiting, Retrying and Oldest wait (the oldest
	unfinished job's age, which says more than how many are waiting: one job stuck for 45 minutes
	matters more than 500 that are 2 minutes old), then the queue split by state. Counted from the
	server's per-state totals; no history (finished jobs leave the queue).
-->
<script lang="ts">
	import Metric from '#lib/components/charts/metric.svelte';
	import ShareBar, { type ShareItem } from '#lib/components/charts/share-bar.svelte';
	import SectionHeading from '#lib/components/section-heading.svelte';
	import { formatSeconds, splitSeconds } from '#lib/dashboard.ts';
	import { jobSubject, secondsUntil, type Names } from '#lib/job.ts';
	import l from '#lib/localization/index.ts';
	import service from '#lib/resources/jobs.ts';

	type Props = { live: boolean; now: number; names: Names; waitingBehind: number };

	let { live, now, names, waitingBehind }: Props = $props();

	const runningQuery = $derived(service.queryAll({ page: 1, status: 'running' }, live));
	const failedQuery = $derived(service.queryAll({ page: 1, status: 'failed' }, live));
	const pendingQuery = $derived(service.queryAll({ page: 1, status: 'pending' }, live));
	const runningStore = $derived(runningQuery.data);
	const failedStore = $derived(failedQuery.data);
	const pendingStore = $derived(pendingQuery.data);
	const running = $derived($runningStore);
	const failed = $derived($failedStore);
	const pending = $derived($pendingStore);

	const label = (job: Parameters<typeof jobSubject>[0]) => {
		const s = jobSubject(job, names);
		return s.environment ? `${s.label} · ${s.environment}` : s.label;
	};

	const oldest = $derived(
		[...(pending?.data ?? []), ...(failed?.data ?? [])].sort((a, b) =>
			a.queued_at.localeCompare(b.queued_at)
		)[0]
	);
	const age = $derived(
		oldest ? Math.max(0, (now - new Date(oldest.queued_at).getTime()) / 1000) : 0
	);
	const ageParts = $derived(oldest ? splitSeconds(age) : ['—', '']);

	const total = $derived((running?.total ?? 0) + (failed?.total ?? 0) + (pending?.total ?? 0));
	const percent = (part: number) => `${total ? Math.round((part / total) * 100) : 0}%`;
	const oldestOf = (jobs: { queued_at: string }[] | undefined) =>
		jobs?.length
			? formatSeconds(
					Math.max(0, (now - Math.min(...jobs.map((j) => new Date(j.queued_at).getTime()))) / 1000)
				)
			: '—';

	const share = $derived<ShareItem[]>([
		{
			key: 'running',
			label: l.translate('status.running'),
			value: running?.total ?? 0,
			color: 'var(--warning)',
			cells: [String(running?.total ?? 0), percent(running?.total ?? 0), oldestOf(running?.data)]
		},
		{
			key: 'failed',
			label: l.translate('jobs.failed_retrying'),
			value: failed?.total ?? 0,
			color: 'var(--danger)',
			cells: [String(failed?.total ?? 0), percent(failed?.total ?? 0), oldestOf(failed?.data)]
		},
		{
			key: 'pending',
			label: l.translate('status.pending'),
			value: pending?.total ?? 0,
			color: 'var(--faint)',
			cells: [String(pending?.total ?? 0), percent(pending?.total ?? 0), oldestOf(pending?.data)]
		}
	]);
</script>

<div class="flex flex-col gap-12">
	<section
		aria-label={l.translate('jobs.figures.label')}
		class="grid grid-cols-2 gap-6 track p-4 sm:gap-x-10 sm:p-5 xl:grid-cols-4"
	>
		<Metric
			label={l.translate('status.running')}
			value={running ? String(running.total) : '—'}
			change={running?.data.length
				? running.data.slice(0, 2).map(label).join(' · ')
				: l.translate('jobs.kpi.running.none')}
		/>
		<Metric
			label={l.translate('jobs.state.waiting')}
			value={pending ? String(pending.total) : '—'}
			change={waitingBehind
				? l.translate('jobs.kpi.waiting.behind', [waitingBehind])
				: l.translate('jobs.kpi.waiting.free')}
		/>
		<Metric
			label={l.translate('jobs.kpi.retrying')}
			value={failed ? String(failed.total) : '—'}
			change={failed?.data.length
				? l.translate('jobs.kpi.retrying.next', [
						label(failed.data[0]),
						secondsUntil(failed.data[0].not_before, now)
					])
				: l.translate('jobs.kpi.retrying.none')}
		/>
		<Metric
			label={l.translate('jobs.kpi.oldest')}
			value={ageParts[0]}
			unit={ageParts[1]}
			change={oldest
				? l.translate('jobs.kpi.oldest.job', [label(oldest), l.time(oldest.queued_at)])
				: l.translate('jobs.kpi.oldest.none')}
		/>
	</section>

	{#if total > 0}
		<section class="flex flex-col gap-3" aria-labelledby="queue-now">
			<SectionHeading id="queue-now">
				{l.translate('jobs.share.title')}
				{#snippet aside()}{l.translate('jobs.share.note', [total])}{/snippet}
			</SectionHeading>
			<div class="track p-4 sm:p-5">
				<ShareBar
					items={share}
					headers={[
						l.translate('jobs.share.state'),
						l.translate('jobs.share.jobs'),
						l.translate('charts.share'),
						l.translate('jobs.kpi.oldest')
					]}
					label={share.map((i) => `${i.label} ${i.value}`).join(', ')}
				/>
			</div>
		</section>
	{/if}
</div>
