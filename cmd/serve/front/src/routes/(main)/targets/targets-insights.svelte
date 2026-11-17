<!--
	The Targets page's figures and spread (DESIGN.md §11.8 Targets): Targets, Ready, Need attention
	and Environments served, then "Environments per target" (counted from the applications already
	loaded) and "Deployments per target" for the last 30 days (one dashboard summary per target; a
	self-hosted install has a handful). Targets being deleted are left out.
-->
<script lang="ts">
	import Layers01Icon from '@hugeicons/core-free-icons/Layers01Icon';
	import Rocket01Icon from '@hugeicons/core-free-icons/Rocket01Icon';
	import { fromStore } from 'svelte/store';
	import Metric from '#lib/components/charts/metric.svelte';
	import ShareBar, { type ShareItem } from '#lib/components/charts/share-bar.svelte';
	import Tile from '#lib/components/dashboard/tile.svelte';
	import l from '#lib/localization/index.ts';
	import type { App } from '#lib/resources/apps.ts';
	import dashboard from '#lib/resources/dashboard.ts';
	import { TargetStatus, type Target } from '#lib/resources/targets.ts';
	import { targetBadge } from '#lib/status.ts';
	import { isLocal, usesOf } from '#lib/target.ts';

	type Props = { targets: Target[]; apps: App[] | undefined };

	let { targets, apps }: Props = $props();

	const active = $derived(targets.filter((t) => !t.cleanup_requested_at));
	const ready = $derived(active.filter((t) => t.state.status === TargetStatus.Ready));
	const configuring = $derived(active.filter((t) => t.state.status === TargetStatus.Configuring));
	const failed = $derived(active.filter((t) => t.state.status === TargetStatus.Failed));
	const local = $derived(active.filter(isLocal).length);
	const uses = $derived(active.map((t) => ({ target: t, uses: apps ? usesOf(apps, t.id) : [] })));
	const served = $derived(uses.reduce((sum, u) => sum + u.uses.length, 0));
	const appsServed = $derived(new Set(uses.flatMap((u) => u.uses.map((x) => x.app.id))).size);

	const summaries = $derived(
		active.map((t) => ({
			target: t,
			summary: fromStore(dashboard.querySummary({ range: 30, target: t.id }).data)
		}))
	);

	const shades = [
		'var(--accent)',
		'color-mix(in oklab, var(--accent) 55%, var(--ink))',
		'color-mix(in oklab, var(--accent) 35%, var(--surface))',
		'var(--faint)',
		'var(--line-strong)'
	];
	const shade = (i: number) => shades[Math.min(i, shades.length - 1)];
	const percent = (part: number, whole: number) =>
		`${whole ? Math.round((part / whole) * 100) : 0}%`;

	const envShare = $derived.by((): ShareItem[] => {
		const sorted = [...uses].sort((a, b) => b.uses.length - a.uses.length);
		return sorted.map((u, i) => ({
			key: u.target.id,
			label: u.target.name,
			value: u.uses.length,
			color: shade(i),
			cells: [
				String(u.uses.length),
				percent(u.uses.length, served),
				targetBadge(u.target.state).label
			]
		}));
	});

	const deployShare = $derived.by((): ShareItem[] => {
		const rows = summaries
			.map((s) => ({ target: s.target, current: s.summary.current?.current }))
			.filter((r) => r.current && r.current.deployments > 0)
			.sort((a, b) => b.current!.deployments - a.current!.deployments);
		const total = rows.reduce((sum, r) => sum + r.current!.deployments, 0);
		return rows.map((r, i) => ({
			key: r.target.id,
			label: r.target.name,
			value: r.current!.deployments,
			color: shade(i),
			cells: [
				String(r.current!.deployments),
				percent(r.current!.deployments, total),
				l.translate('dashboard.env.success', [
					percent(r.current!.succeeded, r.current!.succeeded + r.current!.failed)
				])
			]
		}));
	});
	const deploysLoaded = $derived(summaries.every((s) => s.summary.current));
</script>

<div class="flex flex-col gap-12">
	<section
		aria-label={l.translate('targets.figures.label')}
		class="grid grid-cols-2 gap-6 sm:gap-x-10 xl:grid-cols-4"
	>
		<Metric
			label={l.translate('targets.kpi.count')}
			value={String(active.length)}
			change={l.translate('targets.kpi.count.split', [active.length - local, local])}
		/>
		<Metric
			label={l.translate('targets.kpi.ready')}
			value={String(ready.length)}
			unit={l.translate('targets.kpi.of', [active.length])}
			change={configuring.length
				? l.translate('targets.kpi.ready.some', [configuring.length])
				: l.translate('targets.kpi.ready.all')}
		/>
		<Metric
			label={l.translate('targets.kpi.attention')}
			value={String(failed.length)}
			change={failed.length === 0
				? l.translate('targets.kpi.attention.none')
				: failed.length === 1
					? l.translate('targets.kpi.attention.one', [failed[0].name])
					: l.translate('targets.kpi.attention.more', [failed[0].name, failed.length - 1])}
		/>
		<Metric
			label={l.translate('targets.kpi.envs')}
			value={apps ? String(served) : '—'}
			change={l.translate('targets.kpi.envs.apps', [appsServed])}
		/>
	</section>

	<div class="grid gap-x-10 gap-y-12 lg:grid-cols-2">
		<Tile title={l.translate('targets.share.envs')} icon={Layers01Icon} loading={!apps}>
			{#snippet actions()}{l.translate('targets.share.envs.note')}{/snippet}
			<ShareBar
				items={envShare}
				headers={[
					l.translate('column.target'),
					l.translate('targets.share.envs.count'),
					l.translate('charts.share'),
					l.translate('column.state')
				]}
				label={envShare.map((i) => `${i.label} ${i.value}`).join(', ')}
			/>
		</Tile>
		<Tile title={l.translate('targets.share.deploys')} icon={Rocket01Icon} loading={!deploysLoaded}>
			{#snippet actions()}{l.translate('charts.last_days', [30])}{/snippet}
			{#if deployShare.length}
				<ShareBar
					items={deployShare}
					headers={[
						l.translate('column.target'),
						l.translate('dashboard.kpi.deployments'),
						l.translate('charts.share'),
						l.translate('dashboard.kpi.success')
					]}
					label={deployShare.map((i) => `${i.label} ${i.value}`).join(', ')}
				/>
			{:else}
				<p class="text-label text-muted-foreground">
					{l.translate('dashboard.empty.range.title', [30])}
				</p>
			{/if}
		</Tile>
	</div>
</div>
