<!--
	Activity (DESIGN.md §11.10): deployments across every application, newest first. Older rows load
	with a cursor (Load more), so rows queued meanwhile never shift or repeat a page. When the page
	reports a change (`refresh` increments), the first page is fetched again and merged: new rows go
	on top, known rows take their new state. If the top of the feed is out of view, a pill says how
	many arrived and scrolls back to them.
-->
<script lang="ts">
	import Activity01Icon from '@hugeicons/core-free-icons/Activity01Icon';
	import ArrowDown01Icon from '@hugeicons/core-free-icons/ArrowDown01Icon';
	import ArrowUp01Icon from '@hugeicons/core-free-icons/ArrowUp01Icon';
	import { untrack } from 'svelte';
	import Tile from '#lib/components/dashboard/tile.svelte';
	import Icon from '#lib/components/icon.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { appendFeed, deploymentKey, mergeFeed } from '#lib/dashboard.ts';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service from '#lib/resources/dashboard.ts';
	import type { Deployment, Environment } from '#lib/resources/deployments.ts';
	import { deploymentBadge, deploymentMoment } from '#lib/status.ts';
	import { cn } from '#lib/utils.js';

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

	function source(d: Deployment) {
		if (d.source.discriminator === 'git') {
			return `${d.source.data.branch} · ${d.source.data.hash.slice(0, 7)}`;
		}
		return l.translate(
			d.source.discriminator === 'raw' ? 'deploy.source.raw' : 'deploy.source.archive'
		);
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
		<Button
			size="sm"
			class="sticky top-3 z-10 self-center rounded-full shadow-lg"
			onclick={showNew}
		>
			<Icon icon={ArrowUp01Icon} />
			{l.translate('dashboard.activity.new', [unseen])}
		</Button>
	{/if}

	{#if rows.length > 0}
		<ul class="flex flex-col gap-1.5" aria-live="polite" aria-relevant="additions">
			{#each rows as d (deploymentKey(d))}
				{@const badge = deploymentBadge(d.state)}
				<li
					class={cn(
						'grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-1.5 rounded-md bg-surface py-2.5 pr-3 pl-3 transition-colors duration-700 motion-reduce:transition-none sm:grid-cols-[112px_minmax(0,1fr)_auto]',
						fresh.has(deploymentKey(d)) && 'bg-accent-soft'
					)}
				>
					<span class="max-sm:col-span-2"
						><StatusBadge state={badge.state} label={badge.label} /></span
					>
					<span class="flex min-w-0 flex-col">
						<span class="flex min-w-0 items-center gap-1.5">
							<a
								href={routes.deployment(d.app_id, d.deployment_number)}
								class="truncate text-sm font-medium hover:underline"
							>
								{appName(d.app_id) ?? d.app_id} #{d.deployment_number}
							</a>
							<span
								class={cn(
									'shrink-0 rounded-sm px-1.5 text-[11.5px]/5 font-semibold',
									d.environment === 'production'
										? 'bg-accent-soft text-primary'
										: 'bg-raised text-muted-foreground'
								)}>{d.environment}</span
							>
						</span>
						<span class="truncate text-[12.5px] text-muted-foreground">
							{source(d)} · {d.requested_by.email}
						</span>
					</span>
					<span class="flex flex-col items-end text-[12.5px] text-muted-foreground tabular-nums">
						<span>{l.relative(deploymentMoment(d))}</span>
						<span class="text-faint">
							{d.state.started_at && d.state.finished_at
								? l.duration(d.state.started_at, d.state.finished_at)
								: '—'}
						</span>
					</span>
				</li>
			{/each}
		</ul>
	{:else}
		<p class="flex flex-1 items-center justify-center py-8 text-[13px] text-muted-foreground">
			{l.translate('dashboard.activity.empty')}
		</p>
	{/if}

	{#if cursor}
		<div class="mt-auto flex items-center justify-between gap-3 pt-1 text-[12.5px] text-faint">
			<span>{l.translate('dashboard.activity.showing', [rows.length])}</span>
			<Button variant="outline" size="sm" loading={loadingMore} onclick={loadMore}>
				<Icon icon={ArrowDown01Icon} />
				{l.translate('dashboard.load_more')}
			</Button>
		</div>
	{/if}
</Tile>
