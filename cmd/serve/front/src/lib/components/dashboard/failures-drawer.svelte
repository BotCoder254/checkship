<!--
	Failed deployments of the period (opened from the Failures tile or a bar of the chart), newest
	first with their error code, paged with the activity cursor.
-->
<script lang="ts">
	import ArrowDown01Icon from '@hugeicons/core-free-icons/ArrowDown01Icon';
	import ArrowUpRight01Icon from '@hugeicons/core-free-icons/ArrowUpRight01Icon';
	import { untrack } from 'svelte';
	import Drawer from '#lib/components/drawer.svelte';
	import Icon from '#lib/components/icon.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import { appendFeed, deploymentKey } from '#lib/dashboard.ts';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service, { type DashboardFilters } from '#lib/resources/dashboard.ts';
	import { DeploymentStatus, type Deployment } from '#lib/resources/deployments.ts';
	import { cn } from '#lib/utils.js';

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

	const format = new Intl.DateTimeFormat(l.locale(), { dateStyle: 'medium', timeStyle: 'short' });
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
		<ul class="flex flex-col gap-1.5">
			{#each rows as d (deploymentKey(d))}
				<li>
					<a
						href={routes.deployment(d.app_id, d.deployment_number)}
						class="grid grid-cols-[minmax(0,1fr)_auto] gap-x-3 gap-y-1 rounded-md bg-sunken px-3.5 py-3 transition-colors outline-none hover:bg-hover focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-inset"
					>
						<span class="flex min-w-0 items-center gap-1.5">
							<b class="truncate text-sm font-semibold"
								>{appName(d.app_id) ?? d.app_id} #{d.deployment_number}</b
							>
							<span
								class={cn(
									'shrink-0 rounded-sm px-1.5 text-[11.5px]/5 font-semibold',
									d.environment === 'production'
										? 'bg-accent-soft text-primary'
										: 'bg-raised text-muted-foreground'
								)}>{d.environment}</span
							>
						</span>
						<span class="text-right text-[12.5px] text-muted-foreground">
							{format.format(new Date(d.requested_at))}
						</span>
						<span class="truncate font-mono text-[12.5px] text-danger"
							>{d.state.error_code ?? '—'}</span
						>
						<span class="text-right text-[12.5px] text-faint">
							{d.state.started_at && d.state.finished_at
								? l.duration(d.state.started_at, d.state.finished_at)
								: ''}
						</span>
					</a>
				</li>
			{/each}
		</ul>
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
