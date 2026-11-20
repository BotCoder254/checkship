<!--
	Needs attention: what is broken right now, each with its next step. Production environments whose
	latest deployment failed (Open), targets that failed to configure (Reconfigure) and jobs that
	keep failing (Jobs). Nothing broken says so.
-->
<script lang="ts">
	import Alert02Icon from '@hugeicons/core-free-icons/Alert02Icon';
	import ArrowRight01Icon from '@hugeicons/core-free-icons/ArrowRight01Icon';
	import CancelCircleIcon from '@hugeicons/core-free-icons/CancelCircleIcon';
	import RefreshIcon from '@hugeicons/core-free-icons/RefreshIcon';
	import { toast } from 'svelte-sonner';
	import Tile from '#lib/components/dashboard/tile.svelte';
	import Icon from '#lib/components/icon.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import type { App } from '#lib/resources/apps.ts';
	import { isFailure } from '#lib/status.ts';
	import targetsService, { TargetStatus, type Target } from '#lib/resources/targets.ts';
	import { cn } from '#lib/utils.js';
	import EmptyState from '#lib/components/empty-state.svelte';

	type Props = {
		apps: App[] | undefined;
		targets: Target[] | undefined;
		failingJobs: number | undefined;
		class?: string;
	};

	let { apps, targets, failingJobs, class: className }: Props = $props();

	type Item = {
		key: string;
		tone: 'danger' | 'warning';
		title: string;
		detail: string;
		href?: string;
		action: string;
		reconfigure?: string;
		/** The detail is a code (error), not prose */
		code?: boolean;
	};

	const items = $derived.by(() => {
		const list: Item[] = [];
		for (const app of apps ?? []) {
			const latest = app.latest_deployments.production;
			if (app.cleanup_requested_at || !latest || !isFailure(latest.state)) continue;
			list.push({
				key: `app-${app.id}`,
				tone: 'danger',
				title: l.translate('dashboard.attention.deployment', [app.name, latest.deployment_number]),
				detail: [
					latest.state.error_code,
					l.relative(latest.state.finished_at ?? latest.requested_at)
				]
					.filter(Boolean)
					.join(' · '),
				href: routes.deployment(app.id, latest.deployment_number),
				action: l.translate('dashboard.attention.open'),
				code: true
			});
		}
		for (const target of targets ?? []) {
			if (target.cleanup_requested_at || target.state.status !== TargetStatus.Failed) continue;
			list.push({
				key: `target-${target.id}`,
				tone: 'danger',
				title: l.translate('dashboard.attention.target', [target.name]),
				detail: target.state.error_code ?? '',
				action: l.translate('target.reconfigure'),
				reconfigure: target.id,
				code: true
			});
		}
		if (failingJobs) {
			list.push({
				key: 'jobs',
				tone: 'warning',
				title: l.translate('dashboard.attention.jobs', [failingJobs]),
				detail: l.translate('jobs.retry_note'),
				href: routes.jobs,
				action: l.translate('breadcrumb.jobs')
			});
		}
		return list;
	});

	let busy = $state<string>();

	async function reconfigure(id: string) {
		busy = id;
		try {
			await targetsService.reconfigure(id);
			await targetsService.fetchAll(undefined, { cache: 'no-store' });
		} catch (ex) {
			toast.error(ex instanceof Error ? ex.message : l.translate('unexpected_error'));
		} finally {
			busy = undefined;
		}
	}
</script>

<Tile
	title={l.translate('dashboard.attention')}
	icon={Alert02Icon}
	class={className}
	loading={!apps || !targets}
>
	{#snippet actions()}
		{#if items.length}
			<span class="text-xs text-muted-foreground"
				>{l.translate('dashboard.attention.count', [items.length])}</span
			>
		{/if}
	{/snippet}

	{#if items.length}
		<ul class="-mx-2 flex flex-col gap-0.5">
			{#each items as item (item.key)}
				<li class="grid grid-cols-[32px_minmax(0,1fr)_auto] items-center gap-3 rounded-md p-2">
					<span
						class={cn(
							'flex size-8 items-center justify-center rounded-full',
							item.tone === 'danger' ? 'bg-danger-soft text-danger' : 'bg-warning-soft text-warning'
						)}
					>
						<Icon icon={item.tone === 'danger' ? CancelCircleIcon : Alert02Icon} class="size-4" />
					</span>
					<span class="flex min-w-0 flex-col">
						<span class="text-sm font-medium sm:truncate">{item.title}</span>
						{#if item.detail}
							<span
								class={cn('truncate text-xs text-muted-foreground', item.code && 'font-mono')}
								title={item.detail}>{item.detail}</span
							>
						{/if}
					</span>
					{#if item.reconfigure}
						{@const id = item.reconfigure}
						<Button
							variant="outline"
							size="sm"
							loading={busy === id}
							onclick={() => reconfigure(id)}
						>
							<Icon icon={RefreshIcon} />
							{item.action}
						</Button>
					{:else}
						<Button variant="outline" size="sm" href={item.href}>
							<Icon icon={ArrowRight01Icon} />
							{item.action}
						</Button>
					{/if}
				</li>
			{/each}
		</ul>
	{:else}
		<EmptyState
			illustration="allClear"
			size="compact"
			level={3}
			class="flex-1 justify-center"
			title={l.translate('dashboard.attention.none.title')}
			description={l.translate('dashboard.attention.none')}
		/>
	{/if}
</Tile>
