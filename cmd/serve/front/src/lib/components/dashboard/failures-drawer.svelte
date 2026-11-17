<!--
	Failed deployments of the period (opened from the Failures tile or a bar of the chart), newest
	first with their error code, grouped by day (§5 Lists and tables), paged with the activity
	cursor.
-->
<script lang="ts">
	import ArrowDown01Icon from '@hugeicons/core-free-icons/ArrowDown01Icon';
	import ArrowUpRight01Icon from '@hugeicons/core-free-icons/ArrowUpRight01Icon';
	import DashboardSquare01Icon from '@hugeicons/core-free-icons/DashboardSquare01Icon';
	import ViewIcon from '@hugeicons/core-free-icons/ViewIcon';
	import { untrack } from 'svelte';
	import Drawer from '#lib/components/drawer.svelte';
	import EnvironmentChip from '#lib/components/environment-chip.svelte';
	import Icon from '#lib/components/icon.svelte';
	import TimelineGroup from '#lib/components/timeline/timeline-group.svelte';
	import TimelineRow from '#lib/components/timeline/timeline-row.svelte';
	import { clock, deploymentLabel, deploymentTile } from '#lib/deployment-row.ts';
	import { deploymentMoment } from '#lib/status.ts';
	import { groupByDay } from '#lib/timeline.ts';
	import RowMenuGroup from '#lib/components/table/row-menu-group.svelte';
	import RowMenuItem from '#lib/components/table/row-menu-item.svelte';
	import RowMenu from '#lib/components/table/row-menu.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { appendFeed, deploymentKey } from '#lib/dashboard.ts';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service, { type DashboardFilters } from '#lib/resources/dashboard.ts';
	import { DeploymentStatus, type Deployment } from '#lib/resources/deployments.ts';

	type Props = {
		open: boolean;
		filters: DashboardFilters;
		appName: (id: string) => string | undefined;
	};

	let { open = $bindable(false), filters, appName }: Props = $props();

	let rows = $state<Deployment[]>([]);
	let cursor = $state<string | null>(null);
	let loading = $state(false);
	let error = $state(false);

	async function load(more: boolean) {
		loading = true;
		error = false;
		try {
			const page = await service.fetchActivity({
				range: filters.range,
				environment: filters.environment,
				status: DeploymentStatus.Failed,
				limit: 10,
				cursor: more ? (cursor ?? undefined) : undefined
			});
			rows = more ? appendFeed(rows, page.data) : page.data;
			cursor = page.next_cursor;
		} catch {
			error = true;
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		if (!open) return;
		void filters.range;
		void filters.environment;
		untrack(() => load(false));
	});
</script>

<Drawer
	bind:open
	title={l.translate('dashboard.failures.title')}
	description={l.translate('dashboard.failures.description', [
		filters.range,
		filters.environment
			? l.translate(`deploy.environment.${filters.environment}`)
			: l.translate('dashboard.environment.all')
	])}
>
	{#if error}
		<p class="text-sm text-danger">{l.translate('dashboard.tile.error')}</p>
	{:else if rows.length === 0 && !loading}
		<p class="text-sm text-muted-foreground">{l.translate('dashboard.failures.none')}</p>
	{:else}
		<div class="flex flex-col gap-3">
			{#each groupByDay(rows, deploymentMoment, l.locale()) as group (group.key)}
				<TimelineGroup label={group.label} count={group.items.length}>
					{#each group.items as d (deploymentKey(d))}
						{@const tile = deploymentTile(d.state)}
						{@const name = `${appName(d.app_id) ?? d.app_id} #${d.deployment_number}`}
						<TimelineRow
							icon={tile.icon}
							tone={tile.tone}
							label={deploymentLabel(d, appName(d.app_id))}
							href={routes.deployment(d.app_id, d.deployment_number)}
						>
							{#snippet title()}{name}{/snippet}
							{#snippet badge()}<EnvironmentChip environment={d.environment} />{/snippet}
							{#snippet details()}
								{clock(deploymentMoment(d))}
								{#if d.state.error_code}
									· <span class="font-mono text-danger">{d.state.error_code}</span>
								{/if}
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
									</RowMenuGroup>
								</RowMenu>
							{/snippet}
						</TimelineRow>
					{/each}
				</TimelineGroup>
			{/each}
		</div>
		{#if cursor}
			<div class="flex justify-center">
				<Button variant="outline" size="sm" {loading} onclick={() => load(true)}>
					<Icon icon={ArrowDown01Icon} />
					{l.translate('dashboard.load_more')}
				</Button>
			</div>
		{/if}
	{/if}

	{#snippet footer()}
		<Button variant="outline" href={routes.jobs} class="mr-auto">
			<Icon icon={ArrowUpRight01Icon} />
			{l.translate('dashboard.failures.jobs')}
		</Button>
		<Button variant="outline" onclick={() => (open = false)}>{l.translate('close')}</Button>
	{/snippet}
</Drawer>
