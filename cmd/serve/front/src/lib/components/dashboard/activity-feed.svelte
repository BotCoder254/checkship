<!--
	Activity (DESIGN.md §11.10): deployments across every application, newest first. Older rows load
	with a cursor (Load more), so rows queued meanwhile never shift or repeat a page. When the page
	reports a change (`refresh` increments), the first page is fetched again and merged: new rows go
	on top, known rows take their new state. If the top of the feed is out of view, a pill says how
	many arrived and scrolls back to them.
-->
<script lang="ts">
	import Activity01Icon from '@hugeicons/core-free-icons/Activity01Icon';
	import CheckmarkCircle02Icon from '@hugeicons/core-free-icons/CheckmarkCircle02Icon';
	import Clock01Icon from '@hugeicons/core-free-icons/Clock01Icon';
	import CommandLineIcon from '@hugeicons/core-free-icons/CommandLineIcon';
	import DashboardSquare01Icon from '@hugeicons/core-free-icons/DashboardSquare01Icon';
	import Layers01Icon from '@hugeicons/core-free-icons/Layers01Icon';
	import Rocket01Icon from '@hugeicons/core-free-icons/Rocket01Icon';
	import Timer02Icon from '@hugeicons/core-free-icons/Timer02Icon';
	import ViewIcon from '@hugeicons/core-free-icons/ViewIcon';
	import ArrowDown01Icon from '@hugeicons/core-free-icons/ArrowDown01Icon';
	import ArrowUp01Icon from '@hugeicons/core-free-icons/ArrowUp01Icon';
	import { untrack } from 'svelte';
	import Tile from '#lib/components/dashboard/tile.svelte';
	import Icon from '#lib/components/icon.svelte';
	import EnvironmentChip from '#lib/components/environment-chip.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import RowMenuGroup from '#lib/components/table/row-menu-group.svelte';
	import RowMenuItem from '#lib/components/table/row-menu-item.svelte';
	import RowMenu from '#lib/components/table/row-menu.svelte';
	import * as Table from '#lib/components/ui/table/index.js';
	import { ACTIONS_CELL, ACTIONS_HEAD, TableColumns } from '#lib/table-columns.svelte.ts';
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

	const columns = new TableColumns([
		{
			id: 'status',
			label: l.translate('column.status'),
			icon: CheckmarkCircle02Icon,
			width: 'w-[24%]'
		},
		{
			id: 'deployment',
			label: l.translate('column.deployment'),
			icon: Rocket01Icon,
			width: 'w-[28%]'
		},
		{
			id: 'environment',
			label: l.translate('column.environment'),
			icon: Layers01Icon,
			width: 'w-[18%]',
			priority: 2
		},
		{ id: 'started', label: l.translate('column.started'), icon: Clock01Icon, width: 'w-[16%]' },
		{
			id: 'duration',
			label: l.translate('column.duration'),
			icon: Timer02Icon,
			width: 'w-[14%]',
			priority: 3
		}
	]);

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
		<Table.Root aria-label={l.translate('dashboard.activity')}>
			<Table.Header>
				<Table.Row>
					{#each columns.shown as column (column.id)}
						<Table.Head class={columns.head(column.id)}>{column.label}</Table.Head>
					{/each}
					<Table.Head class={ACTIONS_HEAD}>{l.translate('column.actions')}</Table.Head>
				</Table.Row>
			</Table.Header>
			<Table.Body aria-live="polite" aria-relevant="additions">
				{#each rows as d (deploymentKey(d))}
					{@const badge = deploymentBadge(d.state)}
					{@const name = `${appName(d.app_id) ?? d.app_id} #${d.deployment_number}`}
					<Table.Row
						class={cn(
							'relative *:transition-colors *:duration-700 motion-reduce:*:transition-none',
							fresh.has(deploymentKey(d)) && '*:bg-accent-soft'
						)}
					>
						<Table.Cell class={columns.cell('status')}>
							<StatusBadge state={badge.state} label={badge.label} />
						</Table.Cell>
						<Table.Cell class={columns.cell('deployment')}>
							<a
								href={routes.deployment(d.app_id, d.deployment_number)}
								title={name}
								class="text-sm font-semibold outline-none after:absolute after:inset-0 after:rounded-md focus-visible:after:ring-2 focus-visible:after:ring-ring focus-visible:after:ring-inset"
								>{name}</a
							>
						</Table.Cell>
						<Table.Cell class={columns.cell('environment')}>
							<EnvironmentChip environment={d.environment} />
						</Table.Cell>
						<Table.Cell class={columns.cell('started', 'text-muted-foreground tabular-nums')}>
							{l.relative(deploymentMoment(d))}
						</Table.Cell>
						<Table.Cell class={columns.cell('duration', 'text-muted-foreground tabular-nums')}>
							{d.state.started_at && d.state.finished_at
								? l.duration(d.state.started_at, d.state.finished_at)
								: '—'}
						</Table.Cell>
						<Table.Cell class={ACTIONS_CELL}>
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
						</Table.Cell>
					</Table.Row>
				{/each}
			</Table.Body>
		</Table.Root>
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
