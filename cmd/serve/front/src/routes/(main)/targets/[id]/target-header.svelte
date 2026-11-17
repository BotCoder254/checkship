<!--
	Target header shared by both tabs (DESIGN.md §11.8): the name, a status line (state, where it is,
	who created it), Reconfigure, then the tabs. It never changes between Overview and Settings;
	Settings is a tab, not a second button. Polls the targets list, so a reconfiguration shows up
	without a reload.
-->
<script lang="ts">
	import Loading03Icon from '@hugeicons/core-free-icons/Loading03Icon';
	import RefreshIcon from '@hugeicons/core-free-icons/RefreshIcon';
	import { toast } from 'svelte-sonner';
	import Icon from '#lib/components/icon.svelte';
	import PageHeader from '#lib/components/page-header.svelte';
	import StatusBadge from '#lib/components/status-badge.svelte';
	import { Button } from '#lib/components/ui/button/index.js';
	import l from '#lib/localization/index.ts';
	import routes from '#lib/path.ts';
	import service, { TargetStatus, type Target } from '#lib/resources/targets.ts';
	import { connectionLabel, isLocal } from '#lib/target.ts';
	import { cn } from '#lib/utils.js';
	import TargetTabs from '../target-tabs.svelte';

	type Props = {
		target: Target;
		tab: 'overview' | 'settings';
		/** True while a reconfiguration this page asked for is being sent. */
		reconfiguring?: boolean;
	};

	let { target, tab, reconfiguring = $bindable(false) }: Props = $props();

	const local = $derived(isLocal(target));
	const deleting = $derived(!!target.cleanup_requested_at);
	const targetStatus = $derived(target.state.status);

	/** Ask the server to configure the target again; the failed-state block on Overview uses it too. */
	export async function reconfigure() {
		reconfiguring = true;
		try {
			await service.reconfigure(target.id);
			await service.fetchAll(undefined, { cache: 'no-store' });
		} catch (ex) {
			toast.error(ex instanceof Error ? ex.message : l.translate('unexpected_error'));
		} finally {
			reconfiguring = false;
		}
	}
</script>

<PageHeader
	title={target.name}
	crumbs={[{ href: routes.targets, label: l.translate('breadcrumb.targets') }]}
>
	{#snippet actions()}
		{#if !deleting}
			<Button
				variant="outline"
				loading={reconfiguring}
				disabled={targetStatus === TargetStatus.Configuring}
				onclick={reconfigure}
			>
				<Icon icon={RefreshIcon} />
				{l.translate('target.reconfigure')}
			</Button>
		{/if}
	{/snippet}
	{#snippet status()}
		<span class="flex items-center gap-2 text-ink">
			{#if deleting}
				<StatusBadge state="deleting" label={l.translate('targets.state.deleting')} />
			{:else if targetStatus === TargetStatus.Configuring}
				<Icon icon={Loading03Icon} class="size-3.5 animate-spin text-warning" />
				{l.translate('targets.state.configuring')}
			{:else if targetStatus === TargetStatus.Failed}
				<span class="size-2 rounded-full bg-danger"></span>
				{l.translate('targets.state.failed')}
			{:else}
				<span class="size-2 rounded-full bg-success"></span>
				{target.state.last_ready_version
					? l.translate('targets.state.ready_at', [l.datetime(target.state.last_ready_version)])
					: l.translate('status.ready')}
			{/if}
		</span>
		<span class={cn(!local && 'font-mono text-label')}>{connectionLabel(target)}</span>
		{#if targetStatus === TargetStatus.Failed && !target.state.last_ready_version}
			<span>{l.translate('targets.never_ready')}</span>
		{/if}
		<span>{l.translate('app.created', [l.date(target.created_at), target.created_by.email])}</span>
	{/snippet}
</PageHeader>

<div class="-mt-3 mb-7"><TargetTabs targetId={target.id} active={tab} /></div>
