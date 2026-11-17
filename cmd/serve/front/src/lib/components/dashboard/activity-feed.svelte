<!--
	Activity (DESIGN.md §11.10): deployments across every application, newest first, grouped by day
	(§5 Lists and tables). Older rows load
	with a cursor (Load more), so rows queued meanwhile never shift or repeat a page. When the page
	reports a change (`refresh` increments), the first page is fetched again and merged: new rows go
	on top, known rows take their new state. If the top of the feed is out of view, a pill says how
	many arrived and scrolls back to them.
-->
<script lang="ts">
	import Activity01Icon from '@hugeicons/core-free-icons/Activity01Icon';
	import CommandLineIcon from '@hugeicons/core-free-icons/CommandLineIcon';
	import DashboardSquare01Icon from '@hugeicons/core-free-icons/DashboardSquare01Icon';
	import ViewIcon from '@hugeicons/core-free-icons/ViewIcon';
	import ArrowDown01Icon from '@hugeicons/core-free-icons/ArrowDown01Icon';
	import ArrowUp01Icon from '@hugeicons/core-free-icons/ArrowUp01Icon';
	import { untrack } from 'svelte';
	import Tile from '#lib/components/dashboard/tile.svelte';
	import TimelineGroup from '#lib/components/timeline/timeline-group.svelte';
	import TimelineRow from '#lib/components/timeline/timeline-row.svelte';
	import {
		clock,
		deploymentDuration,
		deploymentLabel,
		deploymentTile
	} from '#lib/deployment-row.ts';
	import { deploymentMoment } from '#lib/status.ts';
	import { groupByDay } from '#lib/timeline.ts';
	import Icon from '#lib/components/icon.svelte';
	import EnvironmentChip from '#lib/components/environment-chip.svelte';
	import RowMenuGroup from '#lib/components/table/row-menu-group.svelte';
	import RowMenuItem from '#lib/components/table/row-menu-item.svelte';
	import RowMenu from '#lib/components/table/row-menu.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { appendFeed, deploymentKey, mergeFeed } from '#lib/dashboard.ts';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service from '#lib/resources/dashboard.ts';
	import type { Deployment, Environment } from '#lib/resources/deployments.ts';

	type Props = {
		environment?: Environment;
		/** Incremented by the page when the server reports a change. */
		refresh: number;
		live: boolean;
		appName: (id: string) => string | undefined;
		class?: string;
	};

	let { environment, refresh, live, appName, class: className }: Props = $props();

	const PAGE = 8;

	let rows = $state<Deployment[]>([]);
	let cursor = $state<string | null>(null);
	let loading = $state(true);
	let loadingMore = $state(false);
	let error = $state<unknown>();
	let fresh = $state(new Set<string>());
	let unseen = $state(0);
	let topVisible = $state(true);
	let top = $state<HTMLElement>();

	async function loadFirst(reset: boolean) {
		try {
			const page = await service.fetchActivity({ environment, limit: PAGE });
			error = undefined;
			if (reset) {
				rows = page.data;
				cursor = page.next_cursor;
				unseen = 0;
				return;
			}
			const merged = mergeFeed(rows, page.data);
			const added = merged.rows.slice(0, merged.added).map(deploymentKey);
			rows = merged.rows;
			if (merged.added) {
				fresh = new Set(added);
				setTimeout(() => (fresh = new Set()), 4000);
				if (!topVisible) unseen += merged.added;
			}
		} catch (ex) {
			if (reset) error = ex;
		} finally {
			loading = false;
		}
	}

	async function loadMore() {
		if (!cursor || loadingMore) return;
		loadingMore = true;
		try {
			const page = await service.fetchActivity({ environment, limit: PAGE, cursor });
			rows = appendFeed(rows, page.data);
			cursor = page.next_cursor;
		} catch (ex) {
			error = ex;
		} finally {
			loadingMore = false;
		}
	}

	// Filters changed: start over.
	$effect(() => {
		void environment;
		untrack(() => {
			loading = true;
			loadFirst(true);
		});
	});

	// A live hint (or the fallback timer): merge the latest rows.
	$effect(() => {
		if (refresh === 0) return;
		untrack(() => loadFirst(false));
	});

	// The sentinel renders once the first page is in.
	$effect(() => {
		if (!top || typeof IntersectionObserver === 'undefined') return;
		const observer = new IntersectionObserver(([entry]) => {
			topVisible = entry.isIntersecting;
			if (topVisible) unseen = 0;
		});
		observer.observe(top);
		return () => observer.disconnect();
	});

	function showNew() {
		top?.scrollIntoView({ behavior: 'smooth', block: 'center' });
		unseen = 0;
	}
</script>

<Tile
	title={l.translate('dashboard.activity')}
	icon={Activity01Icon}
	class={className}
	{loading}
	{error}
	onretry={() => loadFirst(true)}
>
	{#snippet actions()}
		{#if live}
			<span class="flex items-center gap-1.5 text-xs text-success">
				<span class="size-1.5 rounded-full bg-success"></span>
				{l.translate('dashboard.live')}
			</span>
		{/if}
	{/snippet}

	<div bind:this={top} class="-mb-3.5 h-px" aria-hidden="true"></div>
	{#if unseen > 0}
		<Button size="sm" class="sticky top-3 z-10 self-center shadow-lg" onclick={showNew}>
			<Icon icon={ArrowUp01Icon} />
			{l.translate('dashboard.activity.new', [unseen])}
		</Button>
	{/if}

	{#if rows.length > 0}
		<div class="flex flex-col gap-3" aria-live="polite" aria-relevant="additions">
			{#each groupByDay(rows, deploymentMoment, l.locale()) as group (group.key)}
				<TimelineGroup label={group.label} count={group.items.length} sticky={false}>
					{#each group.items as d (deploymentKey(d))}
						{@const tile = deploymentTile(d.state)}
						{@const name = `${appName(d.app_id) ?? d.app_id} #${d.deployment_number}`}
						{@const duration = deploymentDuration(d)}
						<TimelineRow
							icon={tile.icon}
							tone={tile.tone}
							spin={tile.spin}
							label={deploymentLabel(d, appName(d.app_id))}
							href={routes.deployment(d.app_id, d.deployment_number)}
							highlight={fresh.has(deploymentKey(d))}
						>
							{#snippet title()}{name}{/snippet}
							{#snippet badge()}<EnvironmentChip environment={d.environment} />{/snippet}
							{#snippet details()}
								{clock(deploymentMoment(d))}
								{#if d.state.error_code}
									· <span class="font-mono text-danger">{d.state.error_code}</span>
								{/if}
								{#if duration}· {duration}{/if}
							{/snippet}
							{#snippet actions()}
								<RowMenu label={l.translate('table.menu', [name])}>
									<RowMenuGroup>
										<RowMenuItem
											icon={ViewIcon}
											href={routes.deployment(d.app_id, d.deployment_number)}
										>
											{l.translate('jobs.open.deployment')}
										</RowMenuItem>
										<RowMenuItem icon={DashboardSquare01Icon} href={routes.app(d.app_id)}>
											{l.translate('jobs.open.app')}
										</RowMenuItem>
										<RowMenuItem
											icon={CommandLineIcon}
											href={routes.appLogs(d.app_id, d.environment)}
										>
											{l.translate('menu.logs')}
										</RowMenuItem>
									</RowMenuGroup>
								</RowMenu>
							{/snippet}
						</TimelineRow>
					{/each}
				</TimelineGroup>
			{/each}
		</div>
	{:else}
		<p class="flex flex-1 items-center justify-center py-8 text-label text-muted-foreground">
			{l.translate('dashboard.activity.empty')}
		</p>
	{/if}

	{#if cursor}
		<div class="mt-auto flex items-center justify-between gap-3 pt-1 text-xs text-muted-foreground">
			<span>{l.translate('dashboard.activity.showing', [rows.length])}</span>
			<Button variant="outline" size="sm" loading={loadingMore} onclick={loadMore}>
				<Icon icon={ArrowDown01Icon} />
				{l.translate('dashboard.load_more')}
			</Button>
		</div>
	{/if}
</Tile>
